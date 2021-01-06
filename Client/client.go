package main

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	scribble "github.com/nanobox-io/golang-scribble"
)

var serverPath string
var mutex = &sync.Mutex{}

type Fish struct{ Name string }

func makeReq(action string, fname string, from int64, to int64) (ret string) {

	var url strings.Builder
	fmt.Fprintf(&url, "%s%s?from=%d&to=%d&fname=%s", serverPath, action, from, to, fname)

	resp, err := http.Get(url.String())
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	ret = strings.TrimSpace(string(body))
	return
}
func downloadSegment(fname string, from int64, to int64) (ret []byte) {

	var url strings.Builder
	fmt.Fprintf(&url, "%sdownload?from=%d&to=%d&fname=%s", serverPath, from, to, fname)

	resp, err := http.Get(url.String())
	if err != nil {
		log.Fatal("Error connetting")
		return
	}
	if resp.StatusCode != 200 {
		log.Fatal(resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading body")
		return
	}
	ret = body
	return
}

func getSize(fname string) (filesize int64) {
	Size, err := strconv.ParseInt(makeReq("size", fname, 0, 1000), 10, 64)
	if err != nil {
		return
	}
	filesize = Size
	return
}
func getHash(fname string, from int64, to int64) (ret string) {
	ret = makeReq("hash", fname, from, to)
	return
}
func getsha512(bv []byte) (sha string) {
	hasher := sha512.New()
	hasher.Write(bv)
	sha = hex.EncodeToString(hasher.Sum(nil))
	return
}
func checkHashInFileThread(wg *sync.WaitGroup, filename string, from int64, to int64, running *int64) {
	defer func() {
		atomic.AddInt64(running, -1)
		wg.Done()
	}()

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	f.Seek(from, 0)
	r := bufio.NewReader(f)

	b := make([]byte, to)
	n, err := r.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	ret := (b[0:n])
	localHash := getsha512(ret)

	fragmentHash := getHash(filename, from, to)
	if localHash == fragmentHash {
		fmt.Printf("Wow they are the same!\n")
	} else {
		fmt.Printf("Sorry, hashes not equal\n")
		dataSegment := downloadSegment(filename, from, to)
		f.Seek(from, 0)
		f.Write(dataSegment)

	}
}

func checkHashInFile(filename string, from int64, to int64) {

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	f.Seek(from, 0)
	r := bufio.NewReader(f)

	b := make([]byte, to)
	n, err := r.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	ret := (b[0:n])
	localHash := getsha512(ret)

	fragmentHash := getHash(filename, from, to)
	if localHash == fragmentHash {
		fmt.Printf("Wow they are the same!\n")
	} else {
		fmt.Printf("Sorry, hashes not equal\n")
		dataSegment := downloadSegment(filename, from, to)
		f.Seek(from, 0)
		f.Write(dataSegment)

	}
}
func getLocalSize(fname string) (ret int64) {
	ret = 0
	fi, err := os.Stat(fname)
	if err != nil {
		return
	}
	ret = (fi.Size())
	return
}
func resumeFrom(filename string, localSize int64, to int64, repetitions int64, isOdd int64, remoteFileSize int64) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	offset, err := f.Seek(0, 2)
	fmt.Printf("The offset of local file '%s' is %d\nWriting %d times of %d and %d more bytes\n", filename, offset, repetitions, to, isOdd)
	var zeroes []byte
	var i int64
	for i = 0; i < to; i++ {
		zeroes = append(zeroes, 0)
	}
	var repetitionsnow int64 = ((remoteFileSize - localSize) / to)

	for i = 0; i < repetitionsnow; i++ {
		//fmt.Fprintf(f, zeroes)
		mutex.Lock()
		f.Write(zeroes)
		mutex.Unlock()
	}
	isOdd = (remoteFileSize - localSize) % to
	if isOdd > 0 {
		var zeroesFinal []byte
		for i = 0; i < isOdd; i++ {
			zeroesFinal = append(zeroesFinal, 0)
		}
		mutex.Lock()
		f.Write(zeroesFinal)
		mutex.Unlock()
	}
	//f.Close()
	/*for i := 0; i < remoteFileSize-localSize; i++ {
		f.WriteString("a")
	}*/

}
func main() {
	var wg sync.WaitGroup
	var running int64
	var i int64
	var maxRunning int64 = 3
	args := os.Args
	if len(args) < 2 {
		log.Fatal("Argv1 for the server path, argv2 for the filename")
	}
	checkWith := ""
	if len(args) > 2 {
		checkWith = args[3]
	}
	//filename := "ubuntu-20.10-desktop-amd64.iso"
	serverPath = args[1]
	filename := args[2]

	dir := "dbdata"
	db, err := scribble.New(dir, nil)
	if err != nil {
		fmt.Println("Error", err)
	}

	// Write a fish to the database
	fish := Fish{}
	fish.Name = filename

	// Read a fish from the database (passing fish by reference)
	onefish := Fish{}
	if err := db.Read("fish", filename, &onefish); err != nil {
		fmt.Println("FileNotPresentInDatabase", err)
	} else {
		fmt.Println("File just checked before and is perfect")
		log.Fatal(filename)
	}

	remoteFileSize := getSize(filename)
	fmt.Printf("remoteFileSize: %d\n", remoteFileSize)
	if remoteFileSize < 1 {
		log.Fatal("Remote filename not valid (0 bytes)")
	}

	var localSize int64 = getLocalSize(filename)

	var from int64 = 0
	var to int64 = 10 * 1000 * 1024
	var repetitions int64 = ((remoteFileSize) / to)
	lastOdd := (remoteFileSize % to)
	if lastOdd > 0 {
		repetitions++
	}

	if localSize < remoteFileSize {
		fmt.Println("Writing some zeroes to make the same dimension")
		resumeFrom(filename, localSize, to, repetitions, lastOdd, remoteFileSize)

	}

	for i = 0; i < repetitions; i++ {
		from = i * to
		go checkHashInFileThread(&wg, filename, from, to, &running)
		wg.Add(1)
		atomic.AddInt64(&running, 1)
		//checkHashInFile(filename, from, to)
		for atomic.LoadInt64(&running) > maxRunning {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("At %d\n", i)
	}
	fmt.Println("Main: Waiting for workers to finish")
	wg.Wait()
	fmt.Println("Main: Completed")

	os.Truncate(filename, remoteFileSize)

	fmt.Println("File truncated at position like the server")
	cmd := exec.Command("openssl", "sha512", filename)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	opensslHash := string(stdout)
	fragments := strings.Split(opensslHash, ")=")
	if len(fragments) > 1 {
		opensslHash = strings.TrimSpace(fragments[1])
		fmt.Printf("The golang parsed hash: %s\n", opensslHash)
		if len(checkWith) > 0 {
			if checkWith == opensslHash {
				fmt.Printf("Files '%s' perfectly equal now!\n", filename)
				if err := db.Write("fish", filename, fish); err != nil {
					fmt.Println("Error writing to database, expect repetitions", err)
				}
			} else {
				fmt.Printf("Files '%s' are NOT equal. Please check manually\n", filename)
			}
		}
	}
}
