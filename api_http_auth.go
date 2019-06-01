package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

//
// ----------------------
// Session:
// ----------------------
// Session don't exist until after login
//

// Sessions collection of http sessions
// type Sessions struct {
// 	Stores []Session
// }

// HTTPSession is http session
type HTTPSession struct {
	Sessions []Session
}

// SessionValuesType generic values for the session data
type SessionValuesType map[string]interface{}

// Session http session state
type Session struct {
	Expiry time.Time         // expiry time that this cookie becomes invalid
	ID     string            // session id
	Values SessionValuesType // maybe use? (key,value)
}

// ErrSessionExpired session is expired
var ErrSessionExpired = errors.New("expired session")

// find session in request
func (ss *HTTPSession) find(r *http.Request) (*Session, error) {
	// parse cookie
	_ = r.Cookies()
	sid, err := r.Cookie("SessionID")
	if err != nil {
		return nil, err
	}

	for _, s := range ss.Sessions {
		if s.ID == sid.Value {
			return &s, nil
		}
	}

	return nil, ErrSessionExpired
}

// Set session data
func (ss *HTTPSession) Set(r *http.Request, key string, value interface{}) error {
	s, err := ss.find(r)
	if err != nil {
		return err
	}

	if s.Expiry.Before(time.Now()) {
		return ErrSessionExpired
	}

	s.Values[key] = value

	return nil
}

// Get session data
func (ss *HTTPSession) Get(r *http.Request, key string) (interface{}, error) {
	s, err := ss.find(r)
	if err != nil {
		return nil, err
	}

	if s.Expiry.Before(time.Now()) {
		return nil, errors.New("expired session")
	}

	return s.Values[key], nil
}

// Delete force delete session
func (ss *HTTPSession) Delete(r *http.Request, id string) error {
	s, err := ss.find(r)
	if err != nil {
		return err
	}

	s.Expiry = time.Unix(0, 0)

	ss.Scrub()

	return nil
}

// Add new session
func (ss *HTTPSession) Add(r *http.Request, values SessionValuesType) Session {
	sid := generateSessionID(20)

	newSession := Session{
		Expiry: time.Now().AddDate(0, 1, 0), // set default to 1 month expiry
		ID:     sid,
		Values: values,
	}

	ss2 := append(ss.Sessions, newSession)
	ss.Sessions = ss2

	return newSession
}

// Clear all sessions
func (ss *HTTPSession) Clear() {
	ss = nil
}

// Scrub clear all expired sessions
func (ss *HTTPSession) Scrub() {
	nss := []Session{}

	for _, s := range ss.Sessions {
		if s.Expiry.After(time.Now()) {
			nss = append(nss, s)
		}
	}

	ss.Sessions = nss
}

// valid characters for the session id
const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// generateSessionID create random new session string
// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func generateSessionID(n int) string {
	// slightly less deterministic randomness
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[r.Intn(len(letterBytes))]
	}
	return string(b)
}

// login is for basic http login
func login(httpSession *HTTPSession, config *Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		type LoginRequest struct {
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
		strCrypt := SHA256Iter100k(t.Password, config.Salt)
		if subtle.ConstantTimeCompare([]byte(strCrypt), []byte(config.Crypt)) == 1 {
			values := SessionValuesType{
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
func checkLogin(httpSession *HTTPSession, config *Config) func(http.ResponseWriter, *http.Request) {
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
func getPageRoot(httpSession *HTTPSession, config *Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {
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
func getPageMain(httpSession *HTTPSession, config *Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {
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

// CheckAuthHandler is middleware to check and make sure user is logged in
// ref https://cryptic.io/go-http/
func CheckAuthHandler(h http.Handler, httpSession *HTTPSession) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// // TODO remove when done
		// // force auth during dev
		// h.ServeHTTP(w, r)
		// return

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
