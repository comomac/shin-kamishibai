package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	fs := http.FileServer(http.Dir("public/"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	r.Get("/thumbnail/{bookID}", renderThumbnail)
	r.Get("/cbz/{bookID}/{page}", getPage)
	r.Post("/list_dir", listDir)
	r.Get("/bookinfo/{bookID}", bookInfo)
	r.Get("/setbookmark/{bookID}/{page}", setBookmark)
	r.Post("/delete_book", deleteBook)

	log.Fatal(http.ListenAndServe(":8086", r))
}

func bookInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "{}")
}

func setBookmark(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "done")
}

func renderThumbnail(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "bytes")
}

func getPage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "bytes")
}

func listDir(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "{}")
}

func deleteBook(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "deleted")
}
