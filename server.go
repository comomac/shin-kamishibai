package main

import (
	"fmt"
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

	// debug
	fserv := http.FileServer(http.Dir(svr.Config.PathDir + "/web"))

	// generated and packed
	// fs := fileSystem{__binmapName}
	// fserv := http.FileServer(fs)

	h.HandleFunc("/", handlerFS(fserv))

	// public api
	h.HandleFunc("/login", login(httpSession, cfg))

	// private api
	h.HandleFunc("/api/thumbnail/", renderThumbnail(db, cfg)) // /thumbnail/{bookID}
	h.HandleFunc("/api/cbz/", getPageOnly(db))                // /cbz/{bookID}/{page}   get image

	// server side page
	h.HandleFunc("/browse.html", sspBrowse(cfg, db))
	h.HandleFunc("/read.html", sspRead(cfg, db))
	h.HandleFunc("/login.html", sspLogin(cfg, db))

	// middleware
	h1 := CheckAuthHandler(h, httpSession, cfg)

	port := cfg.IP + ":" + strconv.Itoa(cfg.Port)
	fmt.Println("listening on", port)
	fmt.Println("allowed dirs: " + strings.Join(cfg.AllowedDirs, ", "))
	log.Fatal(http.ListenAndServe(port, h1))
}
