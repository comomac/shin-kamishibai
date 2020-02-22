package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Server holds link to database and configuration
type Server struct {
	Database *FlatDB
	Config   *Config
}

// Start launches http server
func (svr *Server) Start() {
	cfg := svr.Config
	db := svr.Database

	// setup session
	httpSession := &SessionStore{}

	h := http.NewServeMux()

	// public folder access

	// debug with local files
	fserv := http.FileServer(http.Dir(svr.Config.PathDir + "/web"))
	fRead := ioutil.ReadFile

	// // use packed file (binfile.go)
	// fs := fakeFileSystem{__binmapName}
	// fRead := fs.ReadFile
	// fserv := http.FileServer(fs)

	h.HandleFunc("/", handlerFS(fserv))

	// public api, page
	h.HandleFunc("/login", loginPOST(httpSession, cfg))
	h.HandleFunc("/login.html", loginGet(cfg, db, fRead))

	// private api, page
	h.HandleFunc("/api/thumbnail/", renderThumbnail(db, cfg)) // /thumbnail/{bookID}    get book cover thumbnail
	h.HandleFunc("/api/read/", readPage(db, true))            // /cbz/{bookID}/{page}   get book page and update last read
	h.HandleFunc("/browse.html", browseGet(cfg, db, fRead, "ssp/browse.ghtml"))
	h.HandleFunc("/legacy.html", browseGet(cfg, db, fRead, "ssp/legacy.ghtml"))
	h.HandleFunc("/read.html", readGet(cfg, db, fRead))

	// middleware
	slog := svrLogging(h, httpSession, cfg)
	h1 := CheckAuthHandler(slog, httpSession, cfg)

	port := cfg.IP + ":" + strconv.Itoa(cfg.Port)
	fmt.Println("listening on", port)
	fmt.Println("allowed dirs: " + strings.Join(cfg.AllowedDirs, ", "))
	log.Fatal(http.ListenAndServe(port, h1))
}
