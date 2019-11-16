package main

import (
	"net/http"
)

// checkLogin so the client knows if current session is login or not
func checkLogin(httpSession *SessionStore, cfg *Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// // TODO remove when done
		// // hack, force auth off during dev
		// w.WriteHeader(http.StatusOK)
		// w.Write([]byte("ok"))
		// return

		// not logged in
		value := httpSession.Get(w, r, "LoggedIn")
		if value != true {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("failed"))
			return
		}

		// logged in
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

// root page
func getPageRoot(httpSession *SessionStore, cfg *Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// // TODO remove when done
		// // hack, force auth off during dev
		// handler.ServeHTTP(w, r)
		// return

		if r.URL.Path == "/" {
			// not logged in, show login page
			value := httpSession.Get(w, r, "LoggedIn")
			if value != true {
				http.Redirect(w, r, "/login.html", http.StatusFound)
				return
			}

			// logged in, show browse
			http.Redirect(w, r, "/browse.html", http.StatusFound)
		} else {
			handler.ServeHTTP(w, r)
		}
		return
	}
}

// browse and tablet page
func getPageMain(httpSession *SessionStore, cfg *Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// // TODO remove when done
		// // hack, force auth off during dev
		// handler.ServeHTTP(w, r)
		// return

		// not logged in, show login page
		value := httpSession.Get(w, r, "LoggedIn")
		if value != true {
			http.Redirect(w, r, "/login.html?referer="+r.URL.Path, http.StatusFound)
			return
		}

		handler.ServeHTTP(w, r)
		return
	}
}
