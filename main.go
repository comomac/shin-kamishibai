package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func startServer() {
	r := chi.NewRouter()

	k := &KRoute{}

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(10 * time.Second))

	fs := http.FileServer(http.Dir("public/"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	// test codes
	r.Get("/cbz/{bookPath}/{page}", k.getPage)
	r.Get("/thumbnail/{bookPath}", k.renderThumbnail)

	// TODO not yet coded
	// r.Get("/thumbnail/{bookID}", renderThumbnail)
	// r.Get("/cbz/{bookID}/{page}", getPage)
	// r.Post("/list_dir", listDir)
	// r.Get("/bookinfo/{bookID}", bookInfo)
	// r.Get("/setbookmark/{bookID}/{page}", setBookmark)
	// r.Post("/delete_book", deleteBook)

	log.Fatal(http.ListenAndServe(":8086", r))
}

func main() {
	// convJtoF()
	// convFtoJ()

	db := NewFlatDB(userHome("etc/shin-kamishibai/db2.txt"))
	db.Load()
	// db.Export(userHome("etc/shin-kamishibai/db2.txt"))
	// ibook := db.IBooks[100]
	// fmt.Printf("%+v %+v\n", ibook, ibook.Book)

	x, err := db.UpdatePage("7IL", 9876)
	check(err)
	fmt.Println(x)

	// db.Export(userHome("etc/shin-kamishibai/db2.txt"))

	// startServer()
}

// 22849
