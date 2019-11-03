package server

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"

	"github.com/comomac/shin-kamishibai/server/pkg/config"
	httpsession "github.com/comomac/shin-kamishibai/server/pkg/httpSession"
	"github.com/comomac/shin-kamishibai/server/pkg/lib"
)

// login is for basic http login
func login(httpSession *httpsession.DataStore, cfg *config.Config) func(http.ResponseWriter, *http.Request) {
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

		decoder := json.NewDecoder(r.Body)
		var t LoginRequest
		err := decoder.Decode(&t)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("cannot parse json data"))
			return
		}

		// w.Header().Set("Content-Type", "application/json")

		// more secure compare
		strCrypt := lib.SHA256Iter(t.Password, cfg.Salt, config.ConfigHashIterations)
		if subtle.ConstantTimeCompare([]byte(strCrypt), []byte(cfg.Crypt)) == 1 {
			values := httpsession.Values{
				"LoggedIn": true,
			}

			newSession := httpSession.Add(r, values)
			// fmt.Printf("sess dat: %+v\n", sess)

			newCookie := &http.Cookie{
				Name:  "SessionID",
				Value: newSession.ID,
			}

			http.SetCookie(w, newCookie)

			// // tablet mode
			// if t.Mode == "tablet" {
			// 	http.Redirect(w, r, "/tablet.html", http.StatusFound)
			// 	return
			// }

			// http.Redirect(w, r, "/browse.html", http.StatusFound)
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("wrong username or password"))
	}
}

// checkLogin so the client knows if current session is login or not
func checkLogin(httpSession *httpsession.DataStore, cfg *config.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// not logged in
		value, err := httpSession.Get(r, "LoggedIn")
		if err != nil || value != true {
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
func getPageRoot(httpSession *httpsession.DataStore, cfg *config.Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Path == "/" {
			// not logged in, show login page
			value, err := httpSession.Get(r, "LoggedIn")
			if err != nil || value != true {
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
func getPageMain(httpSession *httpsession.DataStore, cfg *config.Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// not logged in, show login page
		value, err := httpSession.Get(r, "LoggedIn")
		if err != nil || value != true {
			http.Redirect(w, r, "/login.html?referer="+r.URL.Path, http.StatusFound)
			return
		}

		handler.ServeHTTP(w, r)
		return
	}
}
