package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/comomac/shin-kamishibai/pkg/config"
	"github.com/comomac/shin-kamishibai/pkg/fdb"
	"github.com/comomac/shin-kamishibai/pkg/lib"
)

// FileList contains list of files and folders
type FileList []*FileInfoBasic

// FileInfoBasic basic FileInfo to identify file for dir list
type FileInfoBasic struct {
	IsDir   bool      `json:"is_dir,omitempty"`
	Path    string    `json:"path,omitempty"`     // first item, the current directory
	Name    string    `json:"name,omitempty"`     // file, dir
	ModTime time.Time `json:"mod_time,omitentry"` // file modified time
	More    bool      `json:"more,omitentry"`     // indicate more files behind
	*fdb.Book
}

// dirList lists the folder content, only the folder and the manga will be shown
func dirList(cfg *config.Config, db *fdb.FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()

		dir := query.Get("dir")
		keyword := query.Get("keyword")
		spage := query.Get("page")
		page, err := strconv.Atoi(spage)
		if err != nil {
			page = 1
		}

		fmt.Println("listing dir (", page, ")", dir)

		// check if the dir is allowed to browse
		exists := lib.StringSliceContain(cfg.AllowedDirs, dir)
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

		// add first one as the dir info to save space
		var fileList FileList
		fileList = append(fileList, &FileInfoBasic{
			IsDir: true,
			Path:  dir,
		})

		// fmt.Printf("%+v", db.IMapper)
		// spew.Dump(db.FMapper)

		j := 0
		for _, file := range files {
			if len(keyword) > 0 && !strings.Contains(file.Name(), keyword) {
				continue
			}
			j = j + 1
			if j < (page-1)*ItemsPerPage {
				continue
			}
			if j > (page*ItemsPerPage)-1 {
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

			// not book, skip
			if strings.ToLower(filepath.Ext(file.Name())) != ".cbz" {
				continue
			}

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
			nbook, err := fdb.AddFile(db, fileFullPath)
			if err == nil {
				fib.Book = nbook
				continue
			}

			// error? skip
			fmt.Println("error!", err)
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
