package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/comomac/shin-kamishibai/server/pkg/config"
	"github.com/comomac/shin-kamishibai/server/pkg/fdb"
)

// Server holds link to database and configuration
type Server struct {
	db  fdb.FlatDB
	cfg config.Config
}

// Start launches http server
func Start(cfg *config.Config, db *fdb.FlatDB) {
	// setup session
	httpSession := &HTTPSession{}

	h := http.NewServeMux()

	// public folder access
	fs := http.FileServer(http.Dir("../public"))
	// h.Handle("/", http.StripPrefix("/public/", fs))
	// h.Handle("/", fs)

	// http root path
	h.HandleFunc("/", getPageRoot(httpSession, cfg, fs))

	// direct main page with login follow
	// h.HandleFunc("/tablet.html", getPageMain(httpSession, cfg, fs))
	// h.HandleFunc("/browse.html", getPageMain(httpSession, cfg, fs))
	h.HandleFunc("/browse", browse(httpSession, cfg, fs))

	// public api
	h.HandleFunc("/login", login(httpSession, cfg))

	// private api
	h.HandleFunc("/api/thumbnail/", renderThumbnail(db)) // /thumbnail/{bookID}
	h.HandleFunc("/api/cbz/", getPage(db))               // /cbz/{bookID}/{page}
	h.HandleFunc("/api/bookinfo/", getBookInfo(db))      // /bookinfo/{bookID}
	h.HandleFunc("/api/books_info", getBooksInfo(db))    // /books_info?bookcodes=1,2,3,4,5
	h.HandleFunc("/api/setbookmark/", setBookmark(db))   // /setbookmark/{bookID}/{page}
	h.HandleFunc("/api/lists", getBooksByTitle(db))
	h.HandleFunc("/api/alists", getBooksByAuthor(db))
	h.HandleFunc("/api/list_sources", getSources(cfg))
	h.HandleFunc("/api/lists_dir", dirList(cfg, db))
	h.HandleFunc("/api/check", checkLogin(httpSession, cfg))

	// TODO
	// http.HandleFunc("/alists", postBooksAuthor(db))
	// r.Post("/delete_book", deleteBook)

	// middleware
	// h1 := CheckAuthHandler(h, httpSession)
	// h1 := BasicAuth(h, cfg.Username, cfg.Password, "Authentication required")
	h1 := BasicAuthSession(h, cfg, httpSession, "Authentication required")

	port := ":" + strconv.Itoa(cfg.Port)
	fmt.Println("listening on", port)
	fmt.Println("allowed dirs: " + strings.Join(cfg.AllowedDirs, ", "))
	log.Fatal(http.ListenAndServe(port, h1))
}
