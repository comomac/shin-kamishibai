package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

//
// ----------------------
//  Session
// ----------------------
// Browser session state stored in server memory
//

// SessionStore holds all the http sessions and other details
type SessionStore struct {
	sessions []*session
}

// session http session state
type session struct {
	Expiry time.Time              // expiry time that this cookie becomes invalid
	ID     string                 // session id
	Values map[string]interface{} // holds data in key:value
}

// ErrSessionExpired session is expired
var ErrSessionExpired = errors.New("expired session")

// ErrNoSession session does not exist
var ErrNoSession = errors.New("no session")

// Print hack, print all current sessions
func (ss *SessionStore) Print(r *http.Request) {
	sid, _ := r.Cookie("SessionID")
	fmt.Println("session id", sid)
	for _, s := range ss.sessions {
		fmt.Printf("%+v\n", s)
	}
}

// find session in request
func (ss *SessionStore) find(r *http.Request) (*session, error) {
	// parse cookie
	_ = r.Cookies()
	sid, err := r.Cookie("SessionID")
	if err != nil {
		return nil, err
	}

	for _, s := range ss.sessions {
		if s.ID == sid.Value && s.Expiry.After(time.Now()) {
			return s, nil
		}
	}

	return nil, ErrNoSession
}

// create construct session one session immediately and add to session store
func (ss *SessionStore) create(w http.ResponseWriter, r *http.Request) *session {
	cid := GenerateString(20)

	newSession := &session{
		Expiry: time.Now().AddDate(0, 1, 0), // set default to 1 month expiry
		ID:     cid,
		Values: make(map[string]interface{}),
	}

	ss.sessions = append(ss.sessions, newSession)

	// set browser session cookie
	cki := &http.Cookie{
		Name:  "SessionID",
		Value: newSession.ID,
	}
	http.SetCookie(w, cki)

	return newSession
}

// ready makes sure session exist, if not it will create one on the spot
func (ss *SessionStore) ready(w http.ResponseWriter, r *http.Request) *session {
	ss.Print(r)

	// parse cookie
	_ = r.Cookies()
	_, err := r.Cookie("SessionID")
	if err != nil {
		// no session id in cookie

		// create
		ns := ss.create(w, r)

		return ns
	}

	// find session
	s, err := ss.find(r)
	if err != nil {
		// no session

		// create
		ns := ss.create(w, r)

		return ns
	}

	// existing session
	return s
}

// ID get current session id
func (ss *SessionStore) ID(w http.ResponseWriter, r *http.Request) string {
	s := ss.ready(w, r)

	return s.ID
}

// Set session data
func (ss *SessionStore) Set(w http.ResponseWriter, r *http.Request, key string, value interface{}) {
	s := ss.ready(w, r)

	s.Values[key] = value
}

// Get current session data
func (ss *SessionStore) Get(w http.ResponseWriter, r *http.Request, key string) interface{} {
	s := ss.ready(w, r)

	return s.Values[key]
}

// Delete force delete session
func (ss *SessionStore) Delete(w http.ResponseWriter, r *http.Request) {
	s := ss.ready(w, r)

	s.Expiry = time.Unix(0, 0)

	ss.Scrub()

	// create new session
	ss.create(w, r)
}

// Clear all sessions
func (ss *SessionStore) Clear() {
	ss.sessions = []*session{}
}

// Scrub clear all expired sessions
func (ss *SessionStore) Scrub() {
	nss := []*session{}

	for _, s := range ss.sessions {
		if s.Expiry.After(time.Now()) {
			nss = append(nss, s)
		}
	}

	ss.sessions = nss
}
