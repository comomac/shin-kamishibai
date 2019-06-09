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
	"time"
)

// BookInfoResponse for json response on single book information
type BookInfoResponse struct {
	*Book
	Fullpath Blank `json:"fullpath,omitempty"`
	Inode    Blank `json:"inode,omitempty"`
	Itime    Blank `json:"itime,omitempty"`
}

// BooksInfoResponse for json response on multiple book information
type BooksInfoResponse map[string]*BookInfoResponse

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

// getBooksInfo return several books info
func getBooksInfo(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		bookcodes := r.URL.Query().Get("bookcodes")
		bookIDs := strings.Split(bookcodes, ",")

		books := BooksInfoResponse{}

		for _, bookID := range bookIDs {
			book := db.GetBookByID(bookID)
			if book == nil {
				continue
			}

			books[bookID] = &BookInfoResponse{Book: book}
		}

		b, err := json.Marshal(books)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

// get list of dirs for access
func getSources(config *Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		fmt.Printf("%+v\n", cookies)
		cookie, err := r.Cookie(fmt.Sprintf("%d.order_by", config.Port))
		if err == nil {
			fmt.Printf("%+v %+v\n", cookie.Name, cookie.Value)
		}

		b, err := json.Marshal(config.AllowedDirs)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(b)
	}
}

// post books returns all the book info
func getBooks(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		filterBy := r.URL.Query().Get("filter_by")
		strKeyword := r.URL.Query().Get("keywords")
		keywords := strings.Split(strKeyword, " ")

		books := BooksResponse{}

	OUTER:
		for _, ibook := range db.IBooks {
			// TODO do pageination?
			// if i > 3 {
			// 	break
			// }

			// if there is keyword, make sure title or author matches
			if len(keywords) > 0 {
				for _, keyword := range keywords {
					re := regexp.MustCompile("(?i)" + keyword)
					if re.FindStringIndex(ibook.Title) == nil && re.FindStringIndex(ibook.Author) == nil {
						continue OUTER
					}
				}
			}

			switch filterBy {
			case "finished":
				if ibook.Book.Page == ibook.Book.Pages {
					books = append(books, &BookInfoResponse{Book: ibook.Book})
				}
			case "reading":
				if ibook.Book.Page > 0 && ibook.Book.Page < ibook.Book.Pages {
					books = append(books, &BookInfoResponse{Book: ibook.Book})
				}
			case "new":
				if time.Unix(int64(ibook.Book.Itime)+int64(time.Second)*3600*24*3, 0).Before(time.Now()) {
					books = append(books, &BookInfoResponse{Book: ibook.Book})
				}
			default:
				books = append(books, &BookInfoResponse{Book: ibook.Book})
			}
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

		bookID, page, err := parseURIBookIDandPage(r.RequestURI, "/api/setbookmark/")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		// fmt.Println(r.Method, r.RequestURI, bookID, page)

		ibook := db.MapperID[bookID]
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
		w.Write([]byte("updated"))
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

		ibook := db.MapperID[bookID]
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

		bookID, page, err := parseURIBookIDandPage(r.RequestURI, "/api/cbz/")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		ibook := db.MapperID[bookID]
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

func deleteBook(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "deleted")
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
