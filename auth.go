package main

import (
	"crypto/subtle"
	"net/http"
)

// login is for basic http login
func login(httpSession *SessionStore, cfg *Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Mode     string `json:"mode"`
			Referer  string `json:"referer"`
		}

		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("cannot parse form data"))
			return
		}

		t := LoginRequest{
			Username: r.Form.Get("username"),
			Password: r.Form.Get("password"),
			Mode:     r.Form.Get("mode"),
			Referer:  r.Form.Get("referer"),
		}

		// more secure compare
		strCrypt := SHA256Iter(t.Password, cfg.Salt, ConfigHashIterations)
		if subtle.ConstantTimeCompare([]byte(strCrypt), []byte(cfg.Crypt)) == 1 {
			// create new session
			httpSession.Set(w, r, "LoggedIn", true)

			// // tablet mode
			// if t.Mode == "tablet" {
			// 	http.Redirect(w, r, "/tablet.html", http.StatusFound)
			// 	return
			// }

			http.Redirect(w, r, "/browse.html", http.StatusFound)
			// w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("wrong username or password"))
	}
}
