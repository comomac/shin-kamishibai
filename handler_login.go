package main

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"net/http"
)

// sspLogin login to site
func sspLogin(cfg *Config, db *FlatDB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		query := r.URL.Query()
		referer := query.Get("referer")
		rawQuery := query.Get("rawquery")

		// login template
		data := struct {
			Referer  string
			RawQuery string
		}{
			Referer:  referer,
			RawQuery: rawQuery,
		}
		// helper func for template
		funcMap := template.FuncMap{}
		tmplStr, err := ioutil.ReadFile("ssp/login.ghtml")
		if err != nil {
			responseError(w, err)
			return
		}
		buf := bytes.Buffer{}
		tmpl, err := template.New("login").Funcs(funcMap).Parse(string(tmplStr))
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

	}
}
