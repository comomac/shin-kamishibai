package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// getBooksByAuthor return several books info and group them by book author
func getBooksByAuthor(db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		filterBy := r.URL.Query().Get("filter_by")
		page, _ := strconv.Atoi(r.URL.Query().Get("page")) // pageination, 100 per page
		keywords := r.URL.Query().Get("keywords")

		fmt.Println("filter_by", filterBy, "page", page, "keywords", keywords)

		filtered := filterByAuthor(db.Books, keywords)
		sorted := sortByAuthor(filtered)

		from := 100 * page
		to := 100 * (page + 1)

		if from > len(sorted) {
			// exceeded range
			from = 0
			to = 0
		} else if to > len(sorted)-1 {
			to = len(sorted) - 1
		}

		filtered2 := sorted[from:to]

		books := BooksResponse{}
		for _, book := range filtered2 {
			books = append(books, book)
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
