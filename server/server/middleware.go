package server

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/comomac/shin-kamishibai/server/pkg/config"
	httpsession "github.com/comomac/shin-kamishibai/server/pkg/httpSession"
	"github.com/comomac/shin-kamishibai/server/pkg/lib"
)

// BasicAuth does basic http auth
// ref https://stackoverflow.com/questions/21936332/idiomatic-way-of-requiring-http-basic-auth-in-go
// usage handler := BasicAuth(h, "admin", "123456", "Please enter your username and password for this site")
func BasicAuth(handler http.Handler, username, password, realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// public
		if !strings.Contains(r.URL.Path, "/api/") {
			handler.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()

		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorised.\n"))
			return
		}

		handler.ServeHTTP(w, r)
	}
}

// BasicAuthSession does basic http auth with session support
func BasicAuthSession(handler http.Handler, cfg *config.Config, httpSession *httpsession.DataStore, realm string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// authentication
		if r.URL.Path == "/auth" {
			user, pass, ok := r.BasicAuth()

			// generate crypt hash
			strCrypt := lib.SHA256Iter(pass, cfg.Salt, cfg.Iterations)
			// more secure compare hash
			if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Username)) != 1 || subtle.ConstantTimeCompare([]byte(strCrypt), []byte(cfg.Crypt)) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Unauthorised"))
				return
			}

			// construct new session value
			values := httpsession.Values{
				"LoggedIn": true,
			}
			// remember new session in memory
			newSession := httpSession.Add(r, values)

			// set client session cookie
			newCookie := &http.Cookie{
				Name:  "SessionID",
				Value: newSession.ID,
			}
			http.SetCookie(w, newCookie)

			// force expiring the http basic auth so browser wont remember user/pass
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("welcome"))
			return
		}

		// private

		// get session detail
		value, err := httpSession.Get(r, "LoggedIn")
		if err != nil {
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

		handler.ServeHTTP(w, r)
	}
}

// CheckAuthHandler is middleware to check and make sure user is logged in
// ref https://cryptic.io/go-http/
func CheckAuthHandler(h http.Handler, httpSession *httpsession.DataStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO remove when done
		// hack, force auth during dev
		h.ServeHTTP(w, r)
		return

		// public
		if !strings.Contains(r.URL.Path, "/api/") {
			h.ServeHTTP(w, r)
			return
		}

		// private
		// fmt.Println("api: ", r.URL.Path)

		// get session detail
		value, err := httpSession.Get(r, "LoggedIn")
		if err != nil {
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
	})
}
