package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func browse(httpSession *SessionStore, cfg *Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		files, err := ioutil.ReadDir("./")
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			w.Write([]byte(f.Name()))
		}
	}
}
