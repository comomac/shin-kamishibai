package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type responseErrorStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func responseError(w http.ResponseWriter, err error) {
	resp := &responseErrorStruct{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	}

	str, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, string(str), http.StatusInternalServerError)
	}
}

// Blank use to blank sensitive or not needed data
type Blank string

// BookInfoResponse for json response on book information
type BookInfoResponse struct {
	*Book
	Fullpath Blank `json:"fullpath,omitempty"`
	Inode    Blank `json:"inode,omitempty"`
}

// BooksResponse for json response on all book information
type BooksResponse []*BookInfoResponse

// getBookInfo return indivisual book info
func getBookInfo(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bookID := path.Base(r.RequestURI)

		book := db.GetBookByID(bookID)
		if book == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		b, err := json.Marshal(&BookInfoResponse{Book: book})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

// get books returns all the book info
func getBooks(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		books := BooksResponse{}

		for _, ibook := range db.IBooks {
			// TODO do pageination?
			// if i > 3 {
			// 	break
			// }
			books = append(books, &BookInfoResponse{Book: ibook.Book})
		}

		b, err := json.Marshal(&books)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

// setBookmark remember where the book is read upto
func setBookmark(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bookID, page, err := parseURIBookIDandPage(w, r, "/setbookmark/")
		if err != nil {
			return
		}
		// fmt.Println(r.Method, r.RequestURI, bookID, page)

		ibook := db.IMapper[bookID]
		if ibook == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if uint64(page) > ibook.Pages {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("page cannot be larger than available pages"))
			return
		}

		db.UpdatePage(bookID, page)

		w.WriteHeader(http.StatusOK)
	}
}

// renderThumbnail gives thumbnail on the book
func renderThumbnail(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bookID := path.Base(r.RequestURI)

		ibook := db.IMapper[bookID]
		if ibook == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		fp := ibook.Fullpath

		zr, err := zip.OpenReader(fp)
		if err != nil {
			responseError(w, err)
			return
		}

		regexImageType := regexp.MustCompile(`(?i)\.(jpg|jpeg|gif|png)$`)
		var imgDat []byte
		for _, f := range zr.File {
			if regexImageType.MatchString(f.Name) {
				rc, err := f.Open()
				if err != nil {
					responseError(w, err)
					return
				}
				imgDat, err = imgThumb(rc)
				if err != nil {
					responseError(w, err)
					return
				}
				break
			}
		}

		if len(imgDat) > 0 {
			ctype := http.DetectContentType(imgDat)
			w.Header().Add("Content-Type", ctype)
			w.Header().Add("Content-Length", strconv.Itoa(len(imgDat)))
			w.Write(imgDat)
		} else {
			http.NotFound(w, r)
		}
	}
}

// getPage gives the page of the book
func getPage(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bookID, page, err := parseURIBookIDandPage(w, r, "/cbz/")
		if err != nil {
			return
		}

		ibook := db.IMapper[bookID]
		if ibook == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		fp := ibook.Fullpath

		fmt.Println("file", page, fp)

		if uint64(page) > ibook.Pages {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		zr, err := zip.OpenReader(fp)
		if err != nil {
			responseError(w, err)
			return
		}

		var ttlImages int
		regexImageType := regexp.MustCompile(`(?i)\.(jpg|jpeg|gif|png)$`)
		var imgDat []byte
		var imgFileName string
		for _, f := range zr.File {

			if regexImageType.MatchString(f.Name) {
				// fmt.Println("img!", f.Name)
				ttlImages++
				if page == ttlImages {
					imgFileName = f.Name

					rc, err := f.Open()
					if err != nil {
						responseError(w, err)
						return
					}
					imgDat, err = ioutil.ReadAll(rc)
					if err != nil {
						responseError(w, err)
						return
					}
				}
			} else {
				// fmt.Println("z..", f.Name, f.CompressedSize64, f.UncompressedSize64)
			}
		}

		// fmt.Println("found", ttlImages, "images")
		// fmt.Println(page, "th image name (", imgFileName, ")")

		// fmt.Fprint(w, "bytes")

		if imgFileName == "" {
			http.NotFound(w, r)
			return
		}

		// fmt.Printf("imgDat\n%+v\n", imgDat)
		ctype := http.DetectContentType(imgDat)
		w.Header().Add("Content-Type", ctype)
		w.Header().Add("Content-Length", strconv.Itoa(len(imgDat)))
		w.Write(imgDat)
	}
}

func listDir(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "{}")
}

func deleteBook(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "deleted")
}

// parseURIBookIDandPage parse url and return book id and page. it also do http error if failed
// e.g. /bookinfo/pz3/57    -->    pz3  57
func parseURIBookIDandPage(w http.ResponseWriter, r *http.Request, replStr string) (string, int, error) {
	spt := strings.Split(strings.Replace(r.RequestURI, replStr, "", -1), "/")
	if len(spt) != 2 {
		w.WriteHeader(http.StatusNotFound)
		return "", 0, errors.New("not 2 params")
	}

	bookID := spt[0]
	page, err := strconv.Atoi(spt[1])
	if err != nil {
		errMsg := "invalid page, must be a number"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return "", 0, errors.New(errMsg)
	}
	if page < 0 {
		errMsg := "invalid page, must be 0 or above"
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return "", 0, errors.New(errMsg)
	}

	return bookID, page, nil
}
