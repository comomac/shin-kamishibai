package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func startServer(config *Config, db *FlatDB) {
	// setup session
	httpSession := &HTTPSession{}

	h := http.NewServeMux()

	// public folder access
	fs := http.FileServer(http.Dir("./public"))
	// h.Handle("/", http.StripPrefix("/public/", fs))
	// h.Handle("/", fs)

	// http root path
	h.HandleFunc("/", getRootPage(httpSession, config, fs))

	// public api
	h.HandleFunc("/login", login(httpSession, config))

	// private api
	h.HandleFunc("/api/thumbnail/", renderThumbnail(db)) // /thumbnail/{bookID}
	h.HandleFunc("/api/cbz/", getPage(db))               // /cbz/{bookID}/{page}
	h.HandleFunc("/api/bookinfo/", getBookInfo(db))      // /bookinfo/{bookID}
	h.HandleFunc("/api/books_info", getBooksInfo(db))    // /books_info?bookcodes=1,2,3,4,5
	h.HandleFunc("/api/setbookmark/", setBookmark(db))   // /setbookmark/{bookID}/{page}
	h.HandleFunc("/api/lists", getBooksByTitle(db))
	h.HandleFunc("/api/alists", getBooksByAuther(db))
	h.HandleFunc("/api/list_sources", getSources)
	h.HandleFunc("/api/lists_dir", postDirList(config, db))
	h.HandleFunc("/api/check", checkLogin(httpSession, config))

	// TODO
	// http.HandleFunc("/alists", postBooksAuthor(db))
	// r.Post("/delete_book", deleteBook)

	// middleware
	h1 := CheckAuthHandler(h, httpSession)

	port := ":" + strconv.Itoa(config.Port)
	fmt.Println("listening on", port)
	fmt.Println("allowed dirs: " + strings.Join(config.AllowedDirs, ", "))
	log.Fatal(http.ListenAndServe(port, h1))
}

func main() {
	// fmt.Println(genChar(3))

	// convert format
	// jfile := userHome("etc/kamishibai-kai/db.json")
	// tfile := userHome("etc/shin-kamishibai/db.txt")
	// convJtoF(jfile, tfile) // json to txt
	// convFtoJ(tfile, jfile) // txt to json

	// // load db
	// db := NewFlatDB(userHome("etc/shin-kamishibai/db.txt"))
	// db.Load()

	// fmt.Println(db.BookIDs())
	// fmt.Println(db.GetBookByID("7IL"))

	// // export database, check if it goes generate proper flat db
	// db.Export(userHome("etc/shin-kamishibai/db2.txt"))
	// ibook := db.IBooks[100]
	// fmt.Printf("%+v %+v\n", ibook, ibook.Book)

	// # test if page update works and only update 4 bytes instead of everything
	// x, err := db.UpdatePage("7IL", 9876)
	// check(err)
	// fmt.Println(x)

	config, err := ConfigRead("config.json")
	if err != nil {
		fmt.Println("faile to read config file")
		panic(err)
	}

	// new db
	db := NewFlatDB(userHome("etc/shin-kamishibai/db.txt"))
	db.Load()
	addBooksDir(db, userHome("tmp/mangas"))

	startServer(config, db)
}

// 22849
