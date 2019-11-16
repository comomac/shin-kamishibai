package main

import (
	"fmt"

	"github.com/comomac/shin-kamishibai/pkg/fdb"
)

// used for debug and testing

func main() {
	fmt.Println(GenerateString(3))

	// convert format
	jfile := UserHome("etc/kamishibai-kai/db.json")
	tfile := UserHome("etc/shin-kamishibai/db.txt")
	fdb.ConvJtoF(jfile, tfile) // json to txt
	fdb.ConvFtoJ(tfile, jfile) // txt to json

	// load db
	db := &FlatDB{}
	db.New(tfile)
	db.Load()

	fmt.Println(db.BookIDs())
	fmt.Println(db.GetBookByID("7IL"))

	// export database, check if it goes generate proper flat db
	db.Export(UserHome("etc/shin-kamishibai/db2.txt"))
	ibook := db.IBooks[100]
	fmt.Printf("%+v %+v\n", ibook, ibook.Book)

	// test if page update works and only update 4 bytes instead of everything
	x, err := db.UpdatePage("7IL", 9876)
	if err != nil {
		panic(err)
	}
	fmt.Println(x)
}
