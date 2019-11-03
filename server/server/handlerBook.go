package server

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/comomac/shin-kamishibai/server/pkg/config"
	"github.com/comomac/shin-kamishibai/server/pkg/fdb"
	"github.com/comomac/shin-kamishibai/server/pkg/img"
	"github.com/comomac/shin-kamishibai/server/pkg/lib"
)

// Blank use to blank sensitive or not needed data
type Blank string

// BookInfoResponse for json response on single book information
type BookInfoResponse struct {
	*fdb.Book
	Fullpath Blank `json:"fullpath,omitempty"`
	Inode    Blank `json:"inode,omitempty"`
	Itime    Blank `json:"itime,omitempty"`
}

// BooksInfoResponse for json response on multiple book information
type BooksInfoResponse map[string]*BookInfoResponse

// BooksResponse for json response on all book information
type BooksResponse []*BookInfoResponse

// getBookInfo return indivisual book info
func getBookInfo(db *fdb.FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		items := strings.Split(r.URL.Path, "/")
		bookID := items[len(items)-1]

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
func getBooksInfo(db *fdb.FlatDB) func(http.ResponseWriter, *http.Request) {
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
func getSources(cfg *config.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		fmt.Printf("%+v\n", cookies)
		cookie, err := r.Cookie(fmt.Sprintf("%d.order_by", cfg.Port))
		if err == nil {
			fmt.Printf("%+v %+v\n", cookie.Name, cookie.Value)
		}

		b, err := json.Marshal(cfg.AllowedDirs)
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
func getBooks(db *fdb.FlatDB) func(http.ResponseWriter, *http.Request) {
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
func setBookmark(db *fdb.FlatDB) func(http.ResponseWriter, *http.Request) {
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
func renderThumbnail(db *fdb.FlatDB, cfg *config.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		items := strings.Split(r.URL.Path, "/")
		bookID := items[len(items)-1]

		var imgDat []byte

		// locally stored thumbnail file
		outFile := filepath.Join(filepath.Dir(cfg.Path), "cache", bookID+".jpg")

		isExist, _ := lib.IsFileExists(outFile)
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
		for _, f := range zr.File {
			if regexImageType.MatchString(f.Name) {
				rc, err := f.Open()
				if err != nil {
					responseError(w, err)
					return
				}
				imgDat, err = img.Thumb(rc)
				if err != nil {
					responseError(w, err)
					return
				}
				break
			}
		}

		if len(imgDat) > 0 {
			err2 := ioutil.WriteFile(outFile, imgDat, 0644)
			if err2 != nil {
				fmt.Println("error saving thumbnail", bookID, err2)
			}

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
func getPage(db *fdb.FlatDB) func(http.ResponseWriter, *http.Request) {
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

		fmt.Println("page", page, fp)

		if uint64(page) > ibook.Pages {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		zr, err := zip.OpenReader(fp)
		if err != nil {
			responseError(w, err)
			return
		}
		defer zr.Close()

		regexImageType := regexp.MustCompile(`(?i)\.(jpg|jpeg|gif|png)$`)
		files := []string{}
		for _, f := range zr.File {
			if !regexImageType.MatchString(f.Name) {
				continue
			}

			files = append(files, f.Name)
			// fmt.Println("img!", f.Name)
		}

		// do natural sort
		sort.Slice(files, func(i, j int) bool {
			f1 := regexImageType.ReplaceAllString(files[i], "")
			f2 := regexImageType.ReplaceAllString(files[j], "")
			return lib.AlphaNumCaseCompare(f1, f2)
		})
		// fmt.Println("-------------------------- sorted --------------------------")
		// for _, file := range files {
		// 	fmt.Printf("%+v\n", file)
		// }

		var imgDat []byte // image data to serve

		if page > len(files) {
			responseError(w, errors.New("page beyond file #"))
		}
		getImgFileName := files[page] // image file to get in zip

		if getImgFileName == "" {
			http.NotFound(w, r)
			return
		}

		for _, f := range zr.File {
			if f.Name == getImgFileName {
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
				break
			}
		}

		// fmt.Println("found", ttlImages, "images")
		// fmt.Println(page, "th image name (", imgFileName, ")")

		// fmt.Fprint(w, "bytes")

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
