package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BookInfoByAuthorLite reduced book info to save space and security
type BookInfoByAuthorLite struct {
	*Book
	Author   Blank `json:"author,omitempty"`
	Fullpath Blank `json:"fullpath,omitempty"`
	Inode    Blank `json:"inode,omitempty"`
	Itime    Blank `json:"itime,omitempty"`
}

// BookInfoByAuthorResponse books by single author
type BookInfoByAuthorResponse struct {
	Title  string                  `json:"title"`
	Author string                  `json:"author"`
	Lists  []*BookInfoByAuthorLite `json:"lists"`
}

// BookInfoByAuthorSliceResponse books by multiple authors
type BookInfoByAuthorSliceResponse []*BookInfoByAuthorResponse

// getBooksByAuthor return several books info and group them by book author
func getBooksByAuthor(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		filterBy := r.URL.Query().Get("filter_by")
		page, _ := strconv.Atoi(r.URL.Query().Get("page")) // pageination, 100 per page
		strKeyword := r.URL.Query().Get("keywords")
		keywords := strings.Split(strKeyword, " ")

		books := BookInfoByAuthorSliceResponse{}

		fmt.Println("filter_by", filterBy, "page", page, "keywords", keywords)
		// pageination counter
		i := 0

	OUTER:
		for author, ibooks := range db.MapperAuthor {
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

			// if there is keyword, make sure author or author matches
			if len(keywords) > 0 {
				for _, keyword := range keywords {
					re := regexp.MustCompile("(?i)" + keyword)
					// always assume there is at least 1 book otherwise author wont exists
					if re.FindStringIndex(author) == nil && re.FindStringIndex(ibooks[0].Author) == nil {
						continue OUTER
					}
				}
			}

			gbooks := &BookInfoByAuthorResponse{
				Author: author,
			}

			switch filterBy {
			case "finished":
				for _, ibook := range ibooks {
					if ibook.Book.Page < ibook.Book.Pages {
						continue
					}

					gbooks.Lists = append(gbooks.Lists, &BookInfoByAuthorLite{
						Book: ibook.Book,
					})
				}
			case "reading":
				for _, ibook := range ibooks {
					if ibook.Book.Page == 0 || ibook.Book.Page >= ibook.Book.Pages {
						continue
					}

					gbooks.Lists = append(gbooks.Lists, &BookInfoByAuthorLite{
						Book: ibook.Book,
					})
				}
			case "new":
				for _, ibook := range ibooks {
					if time.Unix(int64(ibook.Book.Itime)+int64(time.Second)*3600*24*3, 0).After(time.Now()) {
						continue
					}

					gbooks.Lists = append(gbooks.Lists, &BookInfoByAuthorLite{
						Book: ibook.Book,
					})
				}
			default:
				for _, ibook := range ibooks {
					gbooks.Lists = append(gbooks.Lists, &BookInfoByAuthorLite{
						Book: ibook.Book,
					})
				}
			}

			if len(gbooks.Lists) > 0 {
				gbooks.Title = gbooks.Lists[0].Title

				// sort by title + volume/chapter (cuz author could have multiple title)
				sort.Slice(gbooks.Lists, func(i, j int) bool {
					a := fmt.Sprintf("%s %s", gbooks.Lists[i].Title, gbooks.Lists[i].Number)
					b := fmt.Sprintf("%s %s", gbooks.Lists[j].Title, gbooks.Lists[j].Number)
					return strings.Compare(a, b) == -1
				})

				books = append(books, gbooks)
			}
		}

		// sort by book authors
		sort.Slice(books, func(i, j int) bool {
			return books[i].Author < books[i].Author
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
