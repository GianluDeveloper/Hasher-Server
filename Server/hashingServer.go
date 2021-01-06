package main

import (
	"bufio"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var WorkingDir string

func getsha512(bv []byte) (sha string) {
	hasher := sha512.New()
	hasher.Write(bv)
	sha = hex.EncodeToString(hasher.Sum(nil))
	return
}
func hashFromFile(fname string, from int64, to int) (ret string) {

	if len(fname) < 1 {
		return
	}
	fullPath := filepath.Join(WorkingDir, fname)
	f, err := os.Open(fullPath)
	if err != nil {
		return
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
		fmt.Println("Error reading file:", err)
		return
	}
	ret = getsha512(b[0:n])

	return
}
func readfromFile(fname string, from int64, to int) (ret []byte) {

	if len(fname) < 1 {
		return
	}
	fullPath := filepath.Join(WorkingDir, fname)
	f, err := os.Open(fullPath)
	if err != nil {
		return
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
		fmt.Println("Error reading file:", err)
		return
	}
	ret = (b[0:n])

	return
}
func gethash(w http.ResponseWriter, req *http.Request) {
	fname := req.URL.Query().Get("fname")
	fname = strings.Replace(fname, "/", "", -1)
	var from int64
	to := 10000

	queryfrom := req.URL.Query().Get("from")

	ffrom, err := strconv.Atoi(queryfrom)
	if err == nil {
		from = int64(ffrom)
	}
	queryto := req.URL.Query().Get("to")

	fto, err := strconv.Atoi(queryto)
	if err == nil {
		to = (fto)
	}

	fmt.Fprintf(w, "%s\n", hashFromFile(fname, from, to))
}
func download(w http.ResponseWriter, req *http.Request) {
	fname := req.URL.Query().Get("fname")
	fname = strings.Replace(fname, "/", "", -1)
	var from int64
	to := 10000

	queryfrom := req.URL.Query().Get("from")

	ffrom, err := strconv.Atoi(queryfrom)
	if err == nil {
		from = int64(ffrom)
	}
	queryto := req.URL.Query().Get("to")

	fto, err := strconv.Atoi(queryto)
	if err == nil {
		to = (fto)
	}
	w.Write(readfromFile(fname, from, to))
}
func getfilesize(w http.ResponseWriter, req *http.Request) {
	fname := req.URL.Query().Get("fname")
	fname = strings.Replace(fname, "/", "", -1)
	fullPath := filepath.Join(WorkingDir, fname)
	fi, err := os.Stat(fullPath)
	if err != nil {
		return
	}
	size := fi.Size()
	fmt.Fprintf(w, "%d", size)
}
func serverHandling(w http.ResponseWriter, req *http.Request) {
	url := req.URL.Path
	fragments := strings.Split(url, "/")
	Func := fragments[len(fragments)-1]
	if Func == "hash" {
		gethash(w, req)
	} else if Func == "size" {
		getfilesize(w, req)
	} else if Func == "download" {
		download(w, req)
	} else {
		fmt.Fprintf(w, "Hello")
	}
	fmt.Println(Func)
}
func main() {
	args := os.Args
	if len(args) < 1 {
		fmt.Println("Not a valid argv1")
	} else {
		WorkingDir = args[1]
		fmt.Println("Starting server at 8080...")

		/*http.HandleFunc("/hash", gethash)
		http.HandleFunc("/size", getfilesize)
		http.HandleFunc("/download", download)*/
		http.HandleFunc("/", serverHandling)
		http.ListenAndServe(":8080", nil)
	}

}
