# Hasher Server written in Go

Server for checking large remote files by chunks with local ones.

Built for transferring TBs of data divided in very large compressed archives (I needed it for ~50gb tgz each) under an unreliable client connection.\
Some data was already there but corrupted somewhere by other download utils. This project was conceived to fix that.

An example of use in a project can be a torrent-like feature, serving chunked contents via HTTP with the speed of Golang backend.

Url parameters for the server:

- **fname** name of the file to work with
- **from** seek at this value (int)
- **to** represent the byte to read after the seek
