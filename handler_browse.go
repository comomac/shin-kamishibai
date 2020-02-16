package main

import (
	"bytes"
	"fmt"
	"html/template"
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
	IsEmpty bool      `json:"is_empty,omitempty"`
	IsBook  bool      `json:"is_book,omitempty"`
	Path    string    `json:"path,omitempty"`     // first item, the current directory
	Name    string    `json:"name,omitempty"`     // file, dir
	ModTime time.Time `json:"mod_time,omitempty"` // file modified time
	More    bool      `json:"more,omitempty"`     // indicate more files behind
	*Book
}

// ItemsPerPage use for pagination
var ItemsPerPage = 18

// browseGet http GET lists the folder content, only the folder and the manga will be shown
func browseGet(cfg *Config, db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()

		dir := query.Get("dir")
		keyword := strings.ToLower(query.Get("keyword"))
		// TODO implement sortBy
		sortBy := strings.ToLower(query.Get("sortby"))
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

		fmt.Println("listing dir (", page, ")", dir)

		// browse template
		data := struct {
			AllowedDirs []string
			Dir         string
			UpDir       string
			Page        int
			Keyword     string
			SortBy      string
			FileList    *FileList
		}{
			AllowedDirs: cfg.AllowedDirs,
			Dir:         dir,
			UpDir:       filepath.Dir(dir),
			Page:        page,
			Keyword:     keyword,
			SortBy:      sortBy,
			FileList:    &FileList{},
		}
		// helper func for template
		funcMap := template.FuncMap{
			"readpc": func(fi *FileInfoBasic) string {
				// read percentage tag
				pg := fi.Page
				pgs := fi.Pages

				r := int(MathRound(float64(pg) / float64(pgs) * 10))
				rr := "read"
				if r == 0 && pg > 0 {
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
		tmplStr, err := ioutil.ReadFile("ssp/browse.ghtml")
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

		// check if the dir is allowed to browse
		exists := StringSliceContain(cfg.AllowedDirs, dir)
		if !exists {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("not allowed to browse " + dir))
			return
		}

		// listing dir
		fileList, err := listDir(dir, keyword, page, db)
		if err != nil {
			responseBadRequest(w, err)
			return
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

func listDir(dir, keyword string, page int, db *FlatDB) (*FileList, error) {
	// listing dir
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
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
	fileList := FileList{}
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
			IsBook:  true,
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

	// nothing found
	if len(fileList) == 1 {
		// add nothing found
		fileList = append(fileList, &FileInfoBasic{
			IsEmpty: true,
		})

		return &fileList, nil
	}

	return &fileList, nil
}
