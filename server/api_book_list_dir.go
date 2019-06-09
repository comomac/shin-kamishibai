package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

// FileList contains list of files and folders
type FileList []*FileInfoBasic

// FileInfoBasic basic FileInfo to identify file for dir list
type FileInfoBasic struct {
	IsDir   bool      `json:"is_dir,omitempty"`
	Path    string    `json:"path,omitempty"`
	Name    string    `json:"name,omitempty"`
	ModTime time.Time `json:"mod_time,omitempty"`
	*Book
}

// dirList lists the folder content, only the folder and the manga will be shown
func dirList(config *Config, db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()

		dir := query.Get("dir")

		fmt.Println(dir)

		// check if the dir is allowed to browse
		exists := StringSliceContain(config.AllowedDirs, dir)
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// listing dir
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		// add first one as the dir info to save space
		var fileList FileList
		fileList = append(fileList, &FileInfoBasic{
			IsDir: true,
			Path:  dir,
		})

		re := regexp.MustCompile(`\.cbz$`)

		// fmt.Printf("%+v", db.IMapper)
		// spew.Dump(db.FMapper)

		for _, file := range files {
			// fmt.Println(file.Name(), file.IsDir())
			if file.IsDir() {
				fileList = append(fileList, &FileInfoBasic{
					IsDir:   true,
					Name:    file.Name(),
					ModTime: file.ModTime(),
				})
			} else if re.MatchString(file.Name()) {
				fib := &FileInfoBasic{
					Name:    file.Name(),
					ModTime: file.ModTime(),
				}

				aaa := dir + "/" + file.Name()
				fmt.Println(aaa)
				book := db.GetBookByPath(aaa)
				if book != nil {
					fib.Book = book
				}

				fileList = append(fileList, fib)
			}
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
