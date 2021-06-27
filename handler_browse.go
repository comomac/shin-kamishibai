package main

import (
	"bytes"
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
	IsDir   bool      `json:"is_dir,omitempty"`   // listing, is it dir?
	IsEmpty bool      `json:"is_empty,omitempty"` // listing, is dir empty?
	IsBook  bool      `json:"is_book,omitempty"`  // listing, is it book?
	Path    string    `json:"path,omitempty"`     // listing, first item - the current directory
	Name    string    `json:"name,omitempty"`     // name of file or dir
	ModTime time.Time `json:"mod_time,omitempty"` // file modified time
	More    bool      `json:"more,omitempty"`     // listing, more files next page
	Book              // not using pointer so can manipulate if necessary
}

// ItemsPerPage use for pagination
var ItemsPerPage = 18

// SortMaxSize maximum allowed size for sorting, otherwise it will skip sort
var SortMaxSize = 300

// special path that is used for special condition for using non-dir path
type specialPath string

const (
	specialPathEveryWhere        specialPath = "__everywhere__"
	specialPathAuthor            specialPath = "__author__"
	specialPathHistory           specialPath = "__history__"
	specialPathHistoryFinished   specialPath = "__history_finished__"
	specialPathHistoryUnfinished specialPath = "__history_unfinished__"
)

func isSpecialPath(dirPath string) bool {
	switch specialPath(dirPath) {
	case specialPathEveryWhere,
		specialPathAuthor,
		specialPathHistory,
		specialPathHistoryFinished,
		specialPathHistoryUnfinished:
		return true
	}
	return false
}

const (
	sortOrderByFileName    = "name"
	sortOrderByFileModTime = "time"
	sortOrderByReadTime    = "read"
	sortOrderByAuthor      = "author"
)

// browseGet http GET lists the folder content, only the folder and the manga will be shown
func browseGet(cfg *Config, db *FlatDB, tmpl *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()

		// if multiple dir parameter is specified, pick by priority
		// order: specialPath... , specialPathEverywhere, dir
		dirs, ok := query["dir"]
		if ok && len(dirs) > 1 {
			for _, x := range dirs {
				switch x {
				case string(specialPathEveryWhere):
					query.Set("dir", string(specialPathEveryWhere))
					break
				}
			}
			http.Redirect(w, r, r.URL.Path+"?"+query.Encode(), http.StatusTemporaryRedirect)
			return
		}

		dir := query.Get("dir")
		keyword := strings.ToLower(query.Get("keyword"))
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

		if isSpecialPath(dir) {
			log.Println("searching (", page, ")", keyword)
		} else {
			log.Println("listing dir (", page, ")", dir)

			// setting up paths, from actual dir to root, reverse order
			// for nav use
			pd := dir
			i := 0 // failsafe
			for i < 15 {
				paths = append(paths, pd)
				if pd == "/" || pd == "." || strings.HasSuffix(pd, ":\\") /* win drive */ {
					break
				}
				pd = filepath.Dir(pd)
				i++
			}
		}

		// browse template struct
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
			Paths:       paths,
			Dir:         dir,
			UpDir:       filepath.Dir(dir),
			Page:        page,
			Keyword:     keyword,
			SortBy:      sortBy,
			FileList:    FileList{},
		}

		buf := bytes.Buffer{}

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

		if isSpecialPath(dir) {
			switch specialPath(dir) {
			case specialPathEveryWhere:
				// tick everywhere checkbox
				data.Everywhere = true

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

			case specialPathAuthor:

				// add first one as the dir info to save space
				fileList = append(fileList, &FileInfoBasic{
					IsDir: true,
					Path:  "Author - " + keyword,
				})

				// build history list
				lstat, lists, err = listByAuthor(db, keyword, page, sortBy)
				if err != nil {
					responseError(w, err)
					return
				}

			case specialPathHistory,
				specialPathHistoryFinished,
				specialPathHistoryUnfinished:

				// add first one as the dir info to save space
				fileList = append(fileList, &FileInfoBasic{
					IsDir: true,
					Path:  "My Read History",
				})

				readState := 0
				switch specialPath(dir) {
				case specialPathHistoryUnfinished:
					readState = 1
				case specialPathHistoryFinished:
					readState = 2
				}

				// build history list
				lstat, lists, err = listByReadHistory(db, keyword, page, readState, sortBy)
				if err != nil {
					responseError(w, err)
					return
				}
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
			lstat, lists, err = listDir(db, dir, keyword, page, sortBy)
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

func listDir(db *FlatDB, dir, search string, page int, sortOrderBy string) (status int, fileList FileList, err error) {
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
		switch sortOrderBy {
		case sortOrderByFileName:
			fileList = sortByFileName(fileList)
		case sortOrderByFileModTime:
			fileList = sortByFileModTime(fileList)
		case sortOrderByReadTime:
			fileList = sortByReadTime(fileList)
		case sortOrderByAuthor:
			fileList = sortByAuthorTitle(fileList)
		default:
			fileList = sortByFileName(fileList)
		}
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
	switch sortOrderBy {
	case sortOrderByFileName:
		fileList = sortByFileName(fileList)
	case sortOrderByFileModTime:
		fileList = sortByFileModTime(fileList)
	case sortOrderByReadTime:
		fileList = sortByReadTime(fileList)
	case sortOrderByAuthor:
		fileList = sortByAuthorTitle(fileList)
	default:
		fileList = sortByFileName(fileList)
	}

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
		// skip if book not exist
		isExist, err := IsFileExists(book.Fullpath)
		if err != nil {
			continue
		}
		if !isExist {
			continue
		}

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

func listByAuthor(db *FlatDB, search string, page int, sortOrderBy string) (status int, fileList FileList, err error) {
	/* status
	-1 error
	 0 no any particular state
	 1 no more list to follow
	 2 more list to follow

	   read state
	0  all
	1  unfinished
	2  finished
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

	// sort by read order
	if len(fileList) <= SortMaxSize {
		switch sortOrderBy {
		case sortOrderByFileName:
			fileList = sortByFileName(fileList)
		case sortOrderByFileModTime:
			fileList = sortByFileModTime(fileList)
		case sortOrderByReadTime:
			fileList = sortByReadTime(fileList)
		default:
			fileList = sortByReadTime(fileList)
		}
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
	switch sortOrderBy {
	case sortOrderByFileName:
		fileList = sortByFileName(fileList)
	case sortOrderByFileModTime:
		fileList = sortByFileModTime(fileList)
	case sortOrderByReadTime:
		fileList = sortByReadTime(fileList)
	default:
		fileList = sortByReadTime(fileList)
	}

	return status, fileList, nil
}

func listByReadHistory(db *FlatDB, search string, page int, readState int, sortOrderBy string) (status int, fileList FileList, err error) {
	/* status
	-1 error
	 0 no any particular state
	 1 no more list to follow
	 2 more list to follow

	   read state
	0  all
	1  unfinished
	2  finished
	*/
	status = -1

	books := db.Search(search)
	for _, book := range books {
		// skip unread books
		if book.Rtime == 0 {
			continue
		}
		switch readState {
		case 1:
			// skip if require unfinish but finished
			if book.Page >= book.Pages {
				continue
			}
		case 2:
			// skip if require finish but unfinished
			if book.Page < book.Pages {
				continue
			}
		}

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

	// sort by read order
	if len(fileList) <= SortMaxSize {
		switch sortOrderBy {
		case sortOrderByFileName:
			fileList = sortByFileName(fileList)
		case sortOrderByFileModTime:
			fileList = sortByFileModTime(fileList)
		case sortOrderByReadTime:
			fileList = sortByReadTime(fileList)
		case sortOrderByAuthor:
			fileList = sortByAuthorTitle(fileList)
		default:
			fileList = sortByReadTime(fileList)
		}
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
	switch sortOrderBy {
	case sortOrderByFileName:
		fileList = sortByFileName(fileList)
	case sortOrderByFileModTime:
		fileList = sortByFileModTime(fileList)
	case sortOrderByReadTime:
		fileList = sortByReadTime(fileList)
	case sortOrderByAuthor:
		fileList = sortByAuthorTitle(fileList)
	default:
		fileList = sortByReadTime(fileList)
	}

	return status, fileList, nil
}
