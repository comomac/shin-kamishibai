package main

import (
	"fmt"
	"log"
	"net/http"
)

func startServer(db *FlatDB) {
	fs := http.FileServer(http.Dir("public/"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	// test codes

	// TODO not yet coded
	http.HandleFunc("/thumbnail/", renderThumbnail(db)) // /thumbnail/{bookID}
	http.HandleFunc("/cbz/", getPage(db))               // /cbz/{bookID}/{page}
	http.HandleFunc("/bookinfo/", getBookInfo(db))      // /bookinfo/{bookID}
	http.HandleFunc("/setbookmark/", setBookmark(db))   // /setbookmark/{bookID}/{page}
	http.HandleFunc("/books", getBooks(db))
	// r.Post("/list_dir", listDir)
	// r.Post("/delete_book", deleteBook)

	port := ":8086"
	fmt.Println("listening on", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func main() {
	// fmt.Println(genChar(3))

	// convert format
	// jfile := userHome("etc/kamishibai-kai/db.json")
	// tfile := userHome("etc/shin-kamishibai/db.txt")
	// convJtoF(jfile, tfile) // json to txt
	// convFtoJ(tfile, jfile) // txt to json

	// load db
	db := NewFlatDB(userHome("etc/shin-kamishibai/db.txt"))
	db.Load()

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

	startServer(db)
}

// 22849
