package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"runtime"
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
	fmt.Println("Starting server")

	cfg := svr.Config
	db := svr.Database

	// setup sessions
	httpSession := &SessionStore{
		serverConfig: cfg,
	}

	// previous sessions
	//err := httpSession.Load()
	//if err != nil {
	//	fmt.Println("Error loading httpSession for previous sessions")
	//	log.Fatal(err)
	//}
	h := http.NewServeMux()

	// public folder access

	// fserv := http.FileServer(http.Dir(svr.Config.PathDir + "/web"))
	// fRead := ioutil.ReadFile

	// use packed file (binfile.go)
	fs := fakeFileSystem{__binmapName}
	// fRead := fs.ReadFile
	fserv := http.FileServer(fs)

	h.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))

	h.HandleFunc("/", handlerFS(fserv))

	// public api, page
	h.HandleFunc("/login", loginPOST(httpSession, cfg))
	h.HandleFunc("/login.html", loginGet(cfg, db, tmplLogin))
	h.HandleFunc("/free", func(w http.ResponseWriter, r *http.Request) {
		runtime.GC()
		w.Write([]byte("freed"))
	})

	// private api, page
	h.HandleFunc("/api/thumbnail/", renderThumbnail(db, cfg)) // /thumbnail/{bookID}              get book cover thumbnail
	h.HandleFunc("/api/read/", readPage(db, true))            // /read?book={bookID}&page={page}  get image and update last read
	h.HandleFunc("/browse.html", browseGet(cfg, db, tmplBrowse))
	h.HandleFunc("/legacy.html", browseGet(cfg, db, tmplBrowseLegacy))
	h.HandleFunc("/read.html", readGet(cfg, db, tmplRead))

	// middleware
	slog := svrLogging(h, httpSession, cfg)
	h1 := CheckAuthHandler(slog, httpSession, cfg)

	port := cfg.IP + ":" + strconv.Itoa(cfg.Port)
	fmt.Println("listening on", port)
	fmt.Println("allowed dirs: " + strings.Join(cfg.AllowedDirs, ", "))
	log.Fatal(http.ListenAndServe(port, h1))
}
