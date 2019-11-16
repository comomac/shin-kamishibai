package main

import (
	"net/http"
	"strings"
)

// LoggedIn keyword for session
var LoggedIn = "LoggedIn"

// CheckAuthHandler is middleware to check and make sure user is logged in
// ref https://cryptic.io/go-http/
func CheckAuthHandler(h http.Handler, httpSession *SessionStore, cfg *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// // TODO remove when done
		// // hack, force auth off during dev
		// h.ServeHTTP(w, r)
		// return

		// fmt.Println("method:", r.Method, "url: ", r.URL.Path)

		// http root path
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/browse.html", http.StatusFound)
			return
		case "/read.html":
			// direct main page with login follow
			getPage(httpSession, cfg, h)
			return
		case "/browse.html":
			getPage(httpSession, cfg, h)
			return
		}

		// private
		if strings.Contains(r.URL.Path, "/api/") {
			// get session detail
			value := httpSession.Get(w, r, LoggedIn)
			if value != true {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorised!"))
				return
			}

			// check if logged in
			if value != true {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Not logged in."))
				return
			}

			h.ServeHTTP(w, r)
			return
		}

		// public
		h.ServeHTTP(w, r)
		return
	})
}
