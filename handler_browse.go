package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
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
	IsEmpty bool      `json:"is_empty,omitempty"`
	IsBook  bool      `json:"is_book,omitempty"`
	Path    string    `json:"path,omitempty"`     // first item, the current directory
	Name    string    `json:"name,omitempty"`     // file, dir
	ModTime time.Time `json:"mod_time,omitempty"` // file modified time
	More    bool      `json:"more,omitempty"`     // indicate more files behind
	Book              // not using pointer so can manipulate if necessary
}

// ItemsPerPage use for pagination
var ItemsPerPage = 18

// SortMaxSize maximum allowed size for sorting, otherwise it will skip sort
var SortMaxSize = 300

// browseGet http GET lists the folder content, only the folder and the manga will be shown
func browseGet(cfg *Config, db *FlatDB, fRead fileReader, htmlTemplateFile string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()

		dir := query.Get("dir")
		keyword := strings.ToLower(query.Get("keyword"))
		// search entire library
		everywhere := strings.ToLower(query.Get("everywhere")) == "true"
		// TODO implement sortBy
		sortBy := strings.ToLower(query.Get("sortby"))
		if sortBy == "" {
			sortBy = "name"
		}
		spage := query.Get("page")
		page, err := strconv.Atoi(spage)
		if err != nil {
			page = 1
		}
		if page < 1 {
			page = 1
		}

		// have clean path, prevent .. bypass
		dir = filepath.Clean(dir)

		// list of path that page can nav up to
		paths := []string{}

		if everywhere {
			log.Println("searching (", page, ")", keyword)
		} else {
			log.Println("listing dir (", page, ")", dir)

			// setting up paths, from actual dir to root, reverse order
			// for nav use
			pd := dir
			i := 0 // failsafe
			for i < 30 {
				paths = append(paths, pd)
				if pd == "/" || pd == "." {
					break
				}
				pd = filepath.Dir(pd)
				i++
			}
		}

		// browse template
		data := struct {
			AllowedDirs []string
			Everywhere  bool
			Paths       []string
			Dir         string
			UpDir       string
			Page        int
			Keyword     string
			SortBy      string
			FileList    FileList
			DirIsMore   bool
			DirIsEmpty  bool
		}{
			AllowedDirs: cfg.AllowedDirs,
			Everywhere:  everywhere,
			Paths:       paths,
			Dir:         dir,
			UpDir:       filepath.Dir(dir),
			Page:        page,
			Keyword:     keyword,
			SortBy:      sortBy,
			FileList:    FileList{},
		}
		// helper func for template
		funcMap := template.FuncMap{
			"dirBase": func(fullpath string) string {
				return filepath.Base(fullpath)
			},
			"readpc": func(fi *FileInfoBasic) string {
				// read percentage tag
				pg := fi.Page
				pgs := fi.Pages

				r := int(MathRound(float64(pg) / float64(pgs) * 10))
				rr := "read"
				if r == 0 && pg > 1 {
					rr += " read5"
				} else if r > 0 {
					rr += fmt.Sprintf(" read%d0", r)
				}
				return rr
			},
			"calcPage": func(a, b int) int {
				c := a + b
				if c < 1 {
					c = 1
				}

				return c
			},
		}
		tmplStr, err := fRead(htmlTemplateFile)
		if err != nil {
			responseError(w, err)
			return
		}
		buf := bytes.Buffer{}
		tmpl, err := template.New("browse").Funcs(funcMap).Parse(string(tmplStr))
		if err != nil {
			responseError(w, err)
			return
		}

		// no dir chosen
		if dir == "" || dir == "." {
			err = tmpl.Execute(&buf, data)
			if err != nil {
				responseError(w, err)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(buf.String()))
			return
		}

		// data to client
		fileList := FileList{}
		var lstat int
		// chopped lists by pagination
		var lists FileList

		if everywhere {
			// add first one as the dir info to save space
			fileList = append(fileList, &FileInfoBasic{
				IsDir: true,
				Path:  "My Entire Library",
			})

			// build library list
			lstat, lists, err = search(db, keyword, page)
			if err != nil {
				responseError(w, err)
				return
			}

		} else {
			// check if the dir is allowed to browse
			exists := StringSliceContain(cfg.AllowedDirs, dir)
			if !exists {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("not allowed to browse " + dir))
				return
			}

			// add first one as the dir info to save space
			fileList = append(fileList, &FileInfoBasic{
				IsDir: true,
				Path:  dir,
			})

			// build dir list
			lstat, lists, err = listDir(db, dir, keyword, page)
			if err != nil {
				responseError(w, err)
				return
			}
		}

		fileList = append(fileList, lists...)

		if lstat == 1 {
			// no more files
			data.DirIsEmpty = true
		}
		if lstat == 2 {
			// more files
			data.DirIsMore = true
		}

		// fill file list data
		data.FileList = fileList
		// exec template
		err = tmpl.Execute(&buf, data)
		if err != nil {
			responseError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(buf.String()))
	}
}

func listDir(db *FlatDB, dir, search string, page int) (status int, fileList FileList, err error) {
	/* status
	-1 error
	 0 no any particular state
	 1 no more list to follow
	 2 more list to follow
	*/
	status = -1

	// listing dir
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return status, nil, err
	}

	search = strings.ToLower(search)
	keywords := strings.Split(search, " ")
	keywords = StringSliceFlatten(keywords)

OUTER:
	for _, file := range files {
		// no dot file/folder
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}

		// case insensitive keyword search
		fulltext := strings.ToLower(file.Name())
		if len(keywords) > 0 {
			found := 0
			for _, keyword := range keywords {
				if !strings.Contains(fulltext, keyword) {
					// no match, next
					continue OUTER
				}
				found++
			}
			if found < len(keywords) {
				// if not all keywords found, skip (doing AND match)
				continue OUTER
			}
		}

		if file.IsDir() {
			// a directory
			fileList = append(fileList, &FileInfoBasic{
				IsDir:   true,
				Name:    file.Name(),
				ModTime: file.ModTime(),
			})

		} else if strings.HasSuffix(fulltext, ".cbz") {
			// a book

			// create and store blank book entry
			fib := &FileInfoBasic{
				IsBook:  true,
				Path:    dir,
				Name:    file.Name(),
				ModTime: file.ModTime(),
			}

			fileList = append(fileList, fib)
		}
	}

	// sort by natural order, if small enough, or lag happens
	if len(fileList) <= SortMaxSize {
		fileList = sortByFileName(fileList)
	}

	// pagination
	head := (page - 1) * ItemsPerPage
	if head > len(fileList) {
		head = len(fileList)
	}
	tail := (page) * ItemsPerPage
	if tail > len(fileList) {
		tail = len(fileList)

		// reached the end, no more files
		status = 1
	} else {
		// indicate more files
		status = 2
	}
	// chopped file list
	fileList = fileList[head:tail]

	// sort again, because earlier sort could be big and skipped
	fileList = sortByFileName(fileList)

	// look up book details
	// doing this way to reduce cpu/disk load, only load the relevant page
	for _, fib := range fileList {
		if !fib.IsBook {
			continue
		}

		// fib.Fullpath is blank, cuz inhertance from blank Book parent
		fileFullPath := fib.Path + "/" + fib.Name

		// find book by path
		book := db.GetBookByPath(fileFullPath)
		if book != nil {
			fib.Book = *book
		} else {
			// book not found, add now
			nbook, err := db.AddFile(fileFullPath)
			if err != nil {
				return status, nil, err
			}
			fib.Book = *nbook

			// clear or eat memory cuz not being used
			fib.Path = ""
		}

		// make page 0 to 1 so wont crash on reading
		if fib.Book.Page <= 0 {
			fib.Book.Page = 1
		}
	}

	return status, fileList, nil
}

func search(db *FlatDB, search string, page int) (status int, fileList FileList, err error) {
	/* status
	-1 error
	 0 no any particular state
	 1 no more list to follow
	 2 more list to follow
	*/
	status = -1

	books := db.Search(search)
	for _, book := range books {
		// create and store blank book entry
		fib := &FileInfoBasic{
			IsBook:  true,
			Name:    filepath.Base(book.Fullpath),
			ModTime: time.Unix(int64(book.Mtime), 0),
			Book:    *book,
		}

		// make page 0 to 1 so wont crash on reading
		if fib.Book.Page <= 0 {
			fib.Book.Page = 1
		}

		fileList = append(fileList, fib)
	}

	// sort by natural order, if small enough, or lag happens
	if len(fileList) <= SortMaxSize {
		fileList = sortByFileName(fileList)
	}

	// pagination
	head := (page - 1) * ItemsPerPage
	if head > len(fileList) {
		head = len(fileList)
	}
	tail := (page) * ItemsPerPage
	if tail > len(fileList) {
		tail = len(fileList)

		// reached the end, no more files
		status = 1
	} else {
		// indicate more files
		status = 2
	}
	// chopped file list
	fileList = fileList[head:tail]

	// sort again, because earlier sort could be big and skipped
	fileList = sortByFileName(fileList)

	return status, fileList, nil
}
