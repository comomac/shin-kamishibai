package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"path"
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
	sessions     []*session
	serverConfig *Config
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

// find session in request
func (ss *SessionStore) find(r *http.Request) (*session, error) {
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
	ss.save()

	// set browser session cookie
	cki := &http.Cookie{
		Name:  "SessionID",
		Value: newSession.ID,
		Path:  "/",
		// commented out so works with go1.8
		// SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cki)
	// remember to stop other .create(), making multiple session id
	r.AddCookie(cki)

	return newSession
}

// ready makes sure session exist, if not it will create one on the spot
func (ss *SessionStore) ready(w http.ResponseWriter, r *http.Request) *session {
	// find cookie
	_, err := r.Cookie("SessionID")
	if err != nil {
		// cookie with session id

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

// saves the sessions from memory into drive for long term storage
func (ss *SessionStore) save() error {
	b := new(bytes.Buffer)
	err := gob.NewEncoder(b).Encode(ss.sessions)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(ss.serverConfig.PathDir, "sessions"), b.Bytes(), 0664)
	if err != nil {
		return err
	}

	return nil
}

// Load previously saved session from drive to memory
func (ss *SessionStore) Load() error {
	f := path.Join(ss.serverConfig.PathDir, "sessions")

	isExist, err := IsFileExists(f)
	if err != nil {
		return err
	}
	if !isExist {
		// previous sessions not exist, continue
		return nil
	}

	b, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}

	bb := bytes.NewBuffer(b)

	err = gob.NewDecoder(bb).Decode(&ss.sessions)
	if err != nil {
		return err
	}
	log.Printf("sessions loaded (%d)\n", len(ss.sessions))
	return nil
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

	ss.save()
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
	ss.save()
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
	ss.save()
}
