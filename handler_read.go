package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
)

// sspRead reads manga
func sspRead(cfg *Config, db *FlatDB) func(http.ResponseWriter, *http.Request) {
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
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("book not found"))
			return
		}
		if page < 1 || page > int(book.Pages) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid page number"))
			return
		}
		// set page temporary
		book.Page = uint64(page)

		// browse template
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
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		buf := bytes.Buffer{}
		tmpl, err := template.New("read").Funcs(funcMap).Parse(string(tmplStr))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		// exec template
		err = tmpl.Execute(&buf, data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(buf.String()))

		// set page read permanently
		db.UpdatePage(bookID, page)

	}
}
