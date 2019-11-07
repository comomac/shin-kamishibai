package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/comomac/shin-kamishibai/pkg/config"
	httpsession "github.com/comomac/shin-kamishibai/pkg/httpSession"
)

func browse(httpSession *httpsession.DataStore, cfg *config.Config, handler http.Handler) func(http.ResponseWriter, *http.Request) {

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
			fmt.Println(f.Name())

			w.Write([]byte(f.Name()))
		}
	}
}
