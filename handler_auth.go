package main

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log"
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
			Referer  string `json:"referer"`
			RawQuery string `json:"rawquery"`
		}

		err := r.ParseForm()
		if err != nil {
			responseError(w, errors.New("cannot parse form data"))
			return
		}

		t := LoginRequest{
			Username: r.Form.Get("username"),
			Password: r.Form.Get("password"),
			Referer:  r.Form.Get("referer"),
			RawQuery: r.Form.Get("rawquery"),
		}

		// more secure compare
		strCrypt := SHA256Iter(t.Password, cfg.Salt, ConfigHashIterations)
		if subtle.ConstantTimeCompare([]byte(strCrypt), []byte(cfg.Crypt)) == 1 {
			// create new session
			httpSession.Set(w, r, LoggedIn, true)

			log.Println("logged in")

			rquery := ""
			origQuery, uerr := base64.URLEncoding.DecodeString(t.RawQuery)
			if uerr == nil {
				rquery = string(origQuery)
			}

			// take referer page if provided
			if len(t.Referer) > 0 {
				http.Redirect(w, r, t.Referer+"?"+rquery, http.StatusFound)
				return
			}

			http.Redirect(w, r, "/browse.html", http.StatusFound)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("wrong username or password"))
	}
}
