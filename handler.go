package main

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

// mimic ioutil.ReadFile
type fileReader func(string) ([]byte, error)

// handle http.FileServer
func handlerFS(handler http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}
}

// get page if login
func getPage(httpSession *SessionStore, cfg *Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// not logged in, show login page
		value := httpSession.Get(w, r, LoggedIn)
		if value != true {
			rawquery := base64.URLEncoding.EncodeToString([]byte(r.URL.RawQuery))

			http.Redirect(w, r, "/login.html?referer="+r.URL.Path+"&rawquery="+rawquery, http.StatusFound)
			return
		}

		// take referer page if provided
		qry := r.URL.Query()
		referer := qry.Get("referer")
		if len(referer) > 0 {
			http.Redirect(w, r, referer, http.StatusFound)
			return
		}

		handler.ServeHTTP(w, r)
		return
	}
}

// helper func for browse, read template
var funcMapBrowse = template.FuncMap{
	"dirBase": func(fullpath string) string {
		// browse, current dir name
		return filepath.Base(fullpath)
	},
	"readpc": func(fi *FileInfoBasic) string {
		// browse, book read percentage tag
		pg := fi.Page
		pgs := fi.Pages

		r := int(MathRound(float64(pg) / float64(pgs) * 10))
		rr := "read"
		if r == 0 && pg > 1 {
			rr += " read5"
		} else if r > 0 {
			rr += fmt.Sprintf(" read%d0", r)
		}
		return rr
	},
	"browsePageN": func(a, b int) int {
		// browse, next or previous listing page
		c := a + b
		if c < 1 {
			return 1
		}
		return c
	},
	"readPageN": func(bk Book, a int) int {
		// readin, for jumping pages
		b := int(bk.Page) + a
		if b < 1 {
			return 1
		}
		if b > int(bk.Pages) {
			return int(bk.Pages)
		}
		return b
	},
}

// prepare templates at start up
var (
	gtmpl            = template.Must(template.New("blank").Funcs(funcMapBrowse).Parse("blank page"))
	tmplBrowse       = template.Must(gtmpl.New("browse").Parse(string(mustRead("ssp/browse.html"))))
	tmplBrowseLegacy = template.Must(gtmpl.New("browseLegacy").Parse(string(mustRead("ssp/legacy.html"))))
	tmplLogin        = template.Must(gtmpl.New("login").Parse(string(mustRead("ssp/login.html"))))
	tmplRead         = template.Must(gtmpl.New("read").Parse(string(mustRead("ssp/read.html"))))
)

func mustRead(filepath string) []byte {
	var fs = fakeFileSystem{__binmapName}
	var fRead = fs.ReadFile
	a, err := fRead(filepath)
	if err != nil {
		log.Fatal(err)
	}
	return a
}
