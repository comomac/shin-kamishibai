package main

import (
	"net/http"
)

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
			http.Redirect(w, r, "/login.html?referer="+r.URL.Path, http.StatusFound)
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
