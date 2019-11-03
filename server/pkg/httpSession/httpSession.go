package httpsession

import (
	"errors"
	"net/http"
	"time"

	"github.com/comomac/shin-kamishibai/server/pkg/lib"
)

//
// ----------------------
//  Session
// ----------------------
// Browser session state stored in server memory
//

// DataStore holds data for http sessions
type DataStore struct {
	Sessions []Session
}

// Values generic key:value for the session data
type Values map[string]interface{}

// Session http session state
type Session struct {
	Expiry time.Time // expiry time that this cookie becomes invalid
	ID     string    // session id
	Values Values    // holds data in key:value
}

// ErrSessionExpired session is expired
var ErrSessionExpired = errors.New("expired session")

// find session in request
func (ss *DataStore) find(r *http.Request) (*Session, error) {
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
func (ss *DataStore) Set(r *http.Request, key string, value interface{}) error {
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
func (ss *DataStore) Get(r *http.Request, key string) (interface{}, error) {
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
func (ss *DataStore) Delete(r *http.Request, id string) error {
	s, err := ss.find(r)
	if err != nil {
		return err
	}

	s.Expiry = time.Unix(0, 0)

	ss.Scrub()

	return nil
}

// Add new session
func (ss *DataStore) Add(r *http.Request, values Values) Session {
	sid := lib.GenerateString(20)

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
func (ss *DataStore) Clear() {
	ss = nil
}

// Scrub clear all expired sessions
func (ss *DataStore) Scrub() {
	nss := []Session{}

	for _, s := range ss.Sessions {
		if s.Expiry.After(time.Now()) {
			nss = append(nss, s)
		}
	}

	ss.Sessions = nss
}
