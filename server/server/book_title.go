package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/comomac/shin-kamishibai/server/pkg/fdb"
)

// BookInfoByTitleLite reduced book info to save space and security
type BookInfoByTitleLite struct {
	*fdb.Book
	Title    Blank `json:"title,omitempty"`
	Fullpath Blank `json:"fullpath,omitempty"`
	Inode    Blank `json:"inode,omitempty"`
	Itime    Blank `json:"itime,omitempty"`
}

// BookInfoByTitleResponse books by single title
type BookInfoByTitleResponse struct {
	Title  string                 `json:"title"`
	Author string                 `json:"author"`
	Lists  []*BookInfoByTitleLite `json:"lists"`
}

// BookInfoByTitleSliceResponse books by multiple titles
type BookInfoByTitleSliceResponse []*BookInfoByTitleResponse

// getBooksByTitle return several books info and group them by book title
func getBooksByTitle(db *fdb.FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		filterBy := r.URL.Query().Get("filter_by")
		page, _ := strconv.Atoi(r.URL.Query().Get("page")) // pageination, 100 per page
		strKeyword := r.URL.Query().Get("keywords")
		keywords := strings.Split(strKeyword, " ")

		books := BookInfoByTitleSliceResponse{}

		fmt.Println("filter_by", filterBy, "page", page, "keywords", keywords)
		// pageination counter
		i := 0

	OUTER:
		for title, ibooks := range db.MapperTitle {
			// pageination
			// 0    1-100
			// 1  101-200
			// 2  201-300
			// 3  301-400
			i++
			if i <= 100*(page) {
				continue
			}
			if i > 100*(page+1) {
				break
			}

			// if there is keyword, make sure title or author matches
			if len(keywords) > 0 {
				for _, keyword := range keywords {
					re := regexp.MustCompile("(?i)" + keyword)
					// always assume there is at least 1 book otherwise title wont exists
					if re.FindStringIndex(title) == nil && re.FindStringIndex(ibooks[0].Author) == nil {
						continue OUTER
					}
				}
			}

			gbooks := &BookInfoByTitleResponse{
				Title: title,
			}

			switch filterBy {
			case "finished":
				for _, ibook := range ibooks {
					if ibook.Book.Page < ibook.Book.Pages {
						continue
					}

					gbooks.Lists = append(gbooks.Lists, &BookInfoByTitleLite{
						Book: ibook.Book,
					})
				}
			case "reading":
				for _, ibook := range ibooks {
					if ibook.Book.Page == 0 || ibook.Book.Page >= ibook.Book.Pages {
						continue
					}

					gbooks.Lists = append(gbooks.Lists, &BookInfoByTitleLite{
						Book: ibook.Book,
					})
				}
			case "new":
				for _, ibook := range ibooks {
					if time.Unix(int64(ibook.Book.Itime)+int64(time.Second)*3600*24*3, 0).After(time.Now()) {
						continue
					}

					gbooks.Lists = append(gbooks.Lists, &BookInfoByTitleLite{
						Book: ibook.Book,
					})
				}
			default:
				for _, ibook := range ibooks {
					gbooks.Lists = append(gbooks.Lists, &BookInfoByTitleLite{
						Book: ibook.Book,
					})
				}
			}

			if len(gbooks.Lists) > 0 {
				gbooks.Author = gbooks.Lists[0].Author

				// sort by volume / chapter
				sort.Slice(gbooks.Lists, func(i, j int) bool {
					return gbooks.Lists[i].Number < gbooks.Lists[j].Number
				})

				books = append(books, gbooks)
			}
		}

		// sort by book titles
		sort.Slice(books, func(i, j int) bool {
			return books[i].Title < books[i].Title
		})

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
