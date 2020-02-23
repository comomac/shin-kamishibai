package main

import (
	"bytes"
	"compress/lzw"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// BinFile copys struct of os.FileInfo to mimic fake file system,
// note another copy of BinFile in generate because go1.4 dont
// support glob in windows 2000, so 2 copies of type BinFile def
type BinFile struct {
	Name         string
	Size         int64
	Mode         os.FileMode
	ModTime      time.Time
	IsDir        bool
	Sys          interface{}
	ready        bool   // is the data filled and ready to serve?
	data         []byte // used during program runtime
	IsCompressed bool   // is it compressed?
	Compression  string // what kind of compression algorithm and setting
	Data64       string // used during generate
}

// Seek whence values
// copied from os package for support deprecated const
const (
	SeekStart   = 0 // seek relative to the origin of the file
	SeekCurrent = 1 // seek relative to the current offset
	SeekEnd     = 2 // seek relative to the end
)

type fakeFileSystem struct {
	BinFileMap map[string]*BinFile
}

func (fsys fakeFileSystem) Open(name string) (http.File, error) {
	f := httpFile{}

	if fsys.BinFileMap[name] == nil {
		return f, os.ErrNotExist
	}
	f.binFile = fsys.BinFileMap[name]

	return f, nil
}

// mimic ioutil.ReadFile
func (fsys fakeFileSystem) ReadFile(name string) ([]byte, error) {
	f := httpFile{}

	if fsys.BinFileMap[name] == nil {
		return nil, os.ErrNotExist
	}
	f.binFile = fsys.BinFileMap[name]

	bdat := make([]byte, f.binFile.Size)
	_, err := f.Read(bdat)
	if err != nil {
		return nil, err
	}

	return bdat, nil
}

// ##############################
// #
// #    mimic http.File
// #
// ##############################

type httpFile struct {
	binFile  *BinFile
	position int64 // cursor pos
}

func (h httpFile) Read(p []byte) (n int, err error) {
	bf := h.binFile
	psize := bf.Size - h.position

	if !bf.ready {
		b, err := base64.URLEncoding.DecodeString(bf.Data64)
		if err != nil {
			log.Println("decode failed", bf.Name)
			return -1, err
		}

		var dat []byte
		if bf.IsCompressed && bf.Compression == "lzw-msb-8" {
			buf := bytes.NewBuffer(b)
			r := lzw.NewReader(buf, lzw.MSB, 8)
			var buf2 bytes.Buffer
			_, err = io.Copy(&buf2, r)
			if err != nil {
				log.Println("decompress copy failed", bf.Name)
				return -1, err
			}
			err = r.Close()
			if err != nil {
				log.Println("decompress close failed", bf.Name)
				return -1, err
			}
			dat = buf2.Bytes()
		} else {
			dat = b
		}

		bf.data = dat
		bf.Data64 = ""
		bf.IsCompressed = false
		bf.ready = true
	}

	copy(p, bf.data[h.position:])

	return int(psize), nil
}
func (h httpFile) Close() error {
	h.position = 0
	return nil
}
func (h httpFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case SeekStart:
		h.position += offset
	case SeekCurrent:
		h.position = offset
	case SeekEnd:
		h.position = h.binFile.Size - offset
	default:
		return -1, errors.New("unknown whence")
	}

	if h.position < 0 {
		return -1, errors.New("pos before 0")
	}
	if h.position >= h.binFile.Size {
		return -1, errors.New("pos past file size")
	}

	return h.position, nil
}
func (h httpFile) Readdir(count int) ([]os.FileInfo, error) {
	// no dir implementation, so return zero length list
	return []os.FileInfo{}, nil
}
func (h httpFile) Stat() (os.FileInfo, error) {
	return fileInfo{h.binFile}, nil
}

// ##############################
// #
// #    mimic FileInfo
// #
// ##############################

type fileInfo struct {
	binFile *BinFile
}

func (f fileInfo) Name() string {
	return f.binFile.Name
}
func (f fileInfo) Size() int64 {
	return f.binFile.Size
}
func (f fileInfo) Mode() os.FileMode {
	return f.binFile.Mode
}
func (f fileInfo) ModTime() time.Time {
	return f.binFile.ModTime
}
func (f fileInfo) IsDir() bool {
	return f.binFile.IsDir
}
func (f fileInfo) Sys() interface{} {
	return nil
}
