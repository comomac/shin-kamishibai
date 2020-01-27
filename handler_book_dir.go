package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FileList contains list of files and folders
type FileList []*FileInfoBasic

// FileInfoBasic basic FileInfo to identify file for dir list
type FileInfoBasic struct {
	IsDir   bool      `json:"is_dir,omitempty"`
	Path    string    `json:"path,omitempty"`     // first item, the current directory
	Name    string    `json:"name,omitempty"`     // file, dir
	ModTime time.Time `json:"mod_time,omitempty"` // file modified time
	More    bool      `json:"more,omitempty"`     // indicate more files behind
	*Book
}

// dirList lists the folder content, only the folder and the manga will be shown
func dirList(cfg *Config, db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()

		dir := query.Get("dir")
		keyword := query.Get("keyword")
		keyword = strings.ToLower(keyword)
		spage := query.Get("page")
		page, err := strconv.Atoi(spage)
		if err != nil {
			page = 1
		}

		// have clean path, prevent .. bypass
		dir = filepath.Clean(dir)

		fmt.Println("listing dir (", page, ")", dir)

		// check if the dir is allowed to browse
		exists := StringSliceContain(cfg.AllowedDirs, dir)
		if !exists {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("not allowed to browse"))
			return
		}

		// listing dir
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		// fmt.Printf("%+v", db.IMapper)
		// spew.Dump(db.FMapper)

		files2 := []os.FileInfo{}
		for _, file := range files {
			// no dot file/folder
			if strings.HasPrefix(file.Name(), ".") {
				continue
			}
			// case insensitive keyword search
			fname := strings.ToLower(file.Name())
			if len(keyword) > 0 && !strings.Contains(fname, keyword) {
				// no match, next
				continue
			}

			if file.IsDir() {
				// a directory
				files2 = append(files2, file)
			} else if strings.ToLower(filepath.Ext(file.Name())) == ".cbz" {
				// a book
				files2 = append(files2, file)
			}
		}

		// add first one as the dir info to save space
		var fileList FileList
		fileList = append(fileList, &FileInfoBasic{
			IsDir: true,
			Path:  dir,
		})

		for i, file := range files2 {
			if i < (page-1)*ItemsPerPage {
				continue
			}
			if i > (page*ItemsPerPage)-1 {
				// indicate more files
				fib := &FileInfoBasic{
					More: true,
				}
				fileList = append(fileList, fib)
				break
			}

			// setup file full path
			fileFullPath := dir + "/" + file.Name()

			// dir
			if file.IsDir() {
				fileList = append(fileList, &FileInfoBasic{
					IsDir:   true,
					Name:    file.Name(),
					ModTime: file.ModTime(),
				})
				continue
			}

			// file

			// create and store blank book entry
			fib := &FileInfoBasic{
				Name:    file.Name(),
				ModTime: file.ModTime(),
			}
			fileList = append(fileList, fib)

			// find book by path
			book := db.GetBookByPath(fileFullPath)
			if book != nil {
				fib.Book = book
				continue
			}

			// find book by name and size
			books := db.SearchBookByNameAndSize(file.Name(), uint64(file.Size()))
			if len(books) > 0 {
				fib.Book = books[0]
				continue
			}

			// book not found, add now
			nbook, err := db.AddFile(fileFullPath)
			if err == nil {
				fib.Book = nbook
				continue
			}

			// error? skip
			fmt.Println("error! bottom fell out", page, keyword, dir)
		}

		b, err := json.Marshal(&fileList)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}
