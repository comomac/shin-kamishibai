package main

import (
	"bytes"
	"errors"
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
