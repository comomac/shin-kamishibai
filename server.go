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
	h.HandleFunc("/check", loginCheck(httpSession, cfg))

	// private api
	h.HandleFunc("/api/thumbnail/", renderThumbnail(db, cfg)) // /thumbnail/{bookID}
	h.HandleFunc("/api/cbz/", getPageOnly(db))                // /cbz/{bookID}/{page}   get image
	h.HandleFunc("/api/read/", getPageNRead(db))              // /read/{bookID}/{page}  get image and update page read
	h.HandleFunc("/api/bookinfo/", getBookInfo(db))           // /bookinfo/{bookID}
	h.HandleFunc("/api/books_info", getBooksInfo(db))         // /books_info?bookcodes=1,2,3,4,5
	h.HandleFunc("/api/setbookmark/", setBookmark(db))        // /setbookmark/{bookID}/{page}
	h.HandleFunc("/api/lists", getBooksByTitle(db))
	h.HandleFunc("/api/alists", getBooksByAuthor(db))
	h.HandleFunc("/api/list_sources", getSources(cfg))
	h.HandleFunc("/api/lists_dir", dirList(cfg, db))

	// server side page
	h.HandleFunc("/ssp/browse.html", sspBrowse(cfg, db))
	h.HandleFunc("/ssp/read.html", sspRead(cfg, db))

	// TODO
	// http.HandleFunc("/alists", postBooksAuthor(db))
	// r.Post("/delete_book", deleteBook)

	// middleware
	h1 := CheckAuthHandler(h, httpSession, cfg)

	port := cfg.IP + ":" + strconv.Itoa(cfg.Port)
	fmt.Println("listening on", port)
	fmt.Println("allowed dirs: " + strings.Join(cfg.AllowedDirs, ", "))
	log.Fatal(http.ListenAndServe(port, h1))
}
