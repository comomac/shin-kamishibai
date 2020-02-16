package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// Blank use to blank sensitive or not needed data
type Blank string

// BooksResponse for json response on multiple books information
type BooksResponse []*Book

// MapBooksResponse string mapped book(s) information
type MapBooksResponse map[string]*Book

// readGet http Get read page
func readGet(cfg *Config, db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()

		bookID := query.Get("book")
		spage := query.Get("page")
		page, err := strconv.Atoi(spage)
		if err != nil {
			page = 1
		}

		book := db.GetBookByID(bookID)
		if book == nil {
			responseBadRequest(w, errors.New("book not found"))
			return
		}
		if page < 1 || page > int(book.Pages) {
			responseBadRequest(w, errors.New("invalid page number"))
			return
		}
		// set page temporary
		book.Page = uint64(page)

		// read template
		data := struct {
			Book *Book
			// Resolution?
		}{
			Book: book,
		}
		// helper func for template
		funcMap := template.FuncMap{
			"calcPage": func(bk Book, a int) int {
				b := int(bk.Page) + a

				if b < 1 {
					b = 1
				}
				if b > int(bk.Pages) {
					b = int(bk.Pages)
				}

				return b
			},
		}
		tmplStr, err := ioutil.ReadFile("ssp/read.ghtml")
		if err != nil {
			responseError(w, err)
			return
		}
		buf := bytes.Buffer{}
		tmpl, err := template.New("read").Funcs(funcMap).Parse(string(tmplStr))
		if err != nil {
			responseError(w, err)
			return
		}

		// exec template
		err = tmpl.Execute(&buf, data)
		if err != nil {
			responseError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(buf.String()))

		// set page read permanently
		db.UpdatePage(bookID, page)

	}
}

// renderThumbnail gives thumbnail on the book
func renderThumbnail(db *FlatDB, cfg *Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		items := strings.Split(r.URL.Path, "/")
		bookID := items[len(items)-1]

		var imgDat []byte

		// check if book is in db
		db.Mutex.Lock()
		ibook := db.MapperID[bookID]
		db.Mutex.Unlock()
		if ibook == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// locally stored thumbnail file
		outFile := filepath.Join(cfg.PathCache, bookID+".jpg")

		// load existing thumbnail
		isExist, _ := IsFileExists(outFile)
		if isExist {
			imgDat, err := ioutil.ReadFile(outFile)
			if err != nil {
				responseError(w, err)
				return
			}

			ctype := http.DetectContentType(imgDat)
			w.Header().Add("Content-Type", ctype)
			w.Header().Add("Content-Length", strconv.Itoa(len(imgDat)))
			w.Write(imgDat)

			return
		}

		zr, err := zip.OpenReader(ibook.Fullpath)
		if err != nil {
			responseError(w, err)
			return
		}
		defer zr.Close()

		// get zip file list
		files := []string{}
		for _, f := range zr.File {
			if !RegexSupportedImageExt.MatchString(f.Name) {
				continue
			}

			files = append(files, f.Name)
		}

		// do natural sort
		sortNatural(files, RegexSupportedImageExt)

		// get first image file
		var rc io.ReadCloser
		for _, f := range zr.File {
			if f.Name != files[0] {
				continue
			}

			// get image data
			rc, err = f.Open()
			if err != nil {
				rc.Close()
				responseError(w, err)
				return
			}
			defer rc.Close()
			break
		}

		// generate thumb
		imgDat, err = ImageThumb(rc)
		if err != nil {
			responseError(w, err)
			return
		}
		if len(imgDat) == 0 {
			responseError(w, errors.New("image length is zero"))
			return
		}

		fmt.Println("created thumbnail", ibook.Fullpath)

		// save thumb
		err2 := ioutil.WriteFile(outFile, imgDat, 0644)
		if err2 != nil {
			fmt.Println("error! failed to save thumbnail", bookID, err2)
		}

		ctype := http.DetectContentType(imgDat)
		w.Header().Add("Content-Type", ctype)
		w.Header().Add("Content-Length", strconv.Itoa(len(imgDat)))
		w.Write(imgDat)
	}
}

// getPageOnly gives the image of the page from the book
func getPageOnly(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bookID, page, err := parseURIBookIDandPage(r.RequestURI, "/api/cbz/")
		if err != nil {
			responseBadRequest(w, err)
			return
		}

		cbzPage(w, r, db, bookID, page, false)
	}
}

// cbzPage shared function for /cbz/... and /read/...
// updatePage == true will update bookmark page
func cbzPage(w http.ResponseWriter, r *http.Request, db *FlatDB, bookID string, pg int, updatePage bool) {
	db.Mutex.Lock()
	ibook := db.MapperID[bookID]
	db.Mutex.Unlock()
	if ibook == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// pg starts at 1 (0 is null)
	// file counter starts at 0. it is still a page, just internal

	fp := ibook.Fullpath

	fmt.Println("page", pg, fp)

	if uint64(pg) > ibook.Pages {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	zr, err := zip.OpenReader(fp)
	if err != nil {
		responseError(w, err)
		return
	}
	defer zr.Close()

	files := []string{}
	for _, f := range zr.File {
		if !RegexSupportedImageExt.MatchString(f.Name) {
			continue
		}

		files = append(files, f.Name)
	}

	// do natural sort
	sortNatural(files, RegexSupportedImageExt)

	// image data to serve
	var imgDat []byte

	if pg > len(files) {
		responseError(w, errors.New("page beyond file #"))
	}
	// image file to get in zip
	getImgFileName := files[pg-1]

	if getImgFileName == "" {
		http.NotFound(w, r)
		return
	}

	for _, f := range zr.File {
		if f.Name != getImgFileName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			responseError(w, err)
			return
		}
		defer rc.Close()

		imgDat, err = ioutil.ReadAll(rc)
		if err != nil {
			responseError(w, err)
			return
		}
		break
	}

	if updatePage {
		// updates bookmark on page read
		db.UpdatePage(bookID, pg)
	}

	ctype := http.DetectContentType(imgDat)
	w.Header().Add("Content-Type", ctype)
	w.Header().Add("Content-Length", strconv.Itoa(len(imgDat)))
	w.Write(imgDat)
}

// parseURIBookIDandPage parse url and return book id and page. it also do http error if failed
// e.g. /bookinfo/pz3/57    -->    pz3  57
// replStr is the text to delete
func parseURIBookIDandPage(uriStr, replStr string) (string, int, error) {
	spt := strings.Split(strings.Replace(uriStr, replStr, "", -1), "/")
	if len(spt) != 2 {
		return "", 0, errors.New("not 2 params")
	}

	bookID := spt[0]
	page, err := strconv.Atoi(spt[1])
	if err != nil {
		return "", 0, errors.New("invalid page, must be a number")
	}
	if page < 0 {
		return "", 0, errors.New("invalid page, must be 0 or above")
	}

	return bookID, page, nil
}
