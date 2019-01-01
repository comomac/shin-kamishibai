package main

// flat file db

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

// FlatDBCharsPage is number of characters reserved for the pages/page
const FlatDBCharsPage = "%04d"

// FlatDBCharsFSize is number of character reserved for the epoch time
const FlatDBCharsFSize = "%010d"

// FlatDBCharsEpoch is number of character reserved for the epoch time
const FlatDBCharsEpoch = "%010d"

// Book contains all the information of book
type Book struct {
	ID       string `json:"id,omitempty"`   // unique id for indexing
	Title    string `json:"title"`          // book title
	Author   string `json:"author"`         // book author, seperated by comma
	Fullpath string `json:"fullpath"`       // book file path
	Ranking  uint64 `json:"ranking"`        // 1-5 ranking, least to most liked
	Fav      uint64 `json:"fav"`            // favourite, 0 false, 1 true
	Cond     uint64 `json:"cond,omitempty"` // 0 unknown, 1 exists, 2 not exist, 3 deleted, 4 inaccessible
	Pages    uint64 `json:"pages"`          // total pages
	Page     uint64 `json:"page"`           // read upto
	Size     uint64 `json:"size"`           // fs file size
	Inode    uint64 `json:"inode"`          // fs inode
	Mtime    uint64 `json:"mtime"`          // fs modified time
	Itime    uint64 `json:"itime"`          // import time
	Rtime    uint64 `json:"rtime"`          // read time
}

// IBook in-memory book, contains extra info on book, used for database
type IBook struct {
	*Book
	Address uint64 `json:"-"`
	Length  uint64 `json:"-"`
}

// Author holds info regards to book
type Author struct {
	Name  string
	Books []*IBook
}

// FlatDB is flat text file database struct
type FlatDB struct {
	IBooks  []*IBook
	Authors []*Author
	Mapper  map[string]*Book
	Path    string // where the database is stored
}

// convert string to uint64
func mustUint64(s string) uint64 {
	i, err := strconv.Atoi(s)
	check(err)
	return uint64(i)
}

func bookCond(fp string) uint64 {
	_, err := os.Stat(fp)
	if err == nil {
		return 1
	}
	if os.IsNotExist(err) {
		return 2
	}
	return 0
}

// NewFlatDB create new Flat Database
func NewFlatDB(params ...string) *FlatDB {
	// default path
	dbPath := "./db.txt"

	if len(params) == 1 {
		dbPath = params[0]
	} else if len(params) > 1 {
		log.Fatal("Too many parameters for NewFlatDB!")
	}

	db := &FlatDB{}
	db.Path = dbPath
	db.Mapper = make(map[string]*Book)

	return db
}

// Clear all data
func (db *FlatDB) Clear() {
	db.IBooks = nil
	db.Authors = nil
	db.Mapper = make(map[string]*Book)
}

// Load data using default file path
func (db *FlatDB) Load() {
	db.Import(db.Path)
}

// Import data from alternative path
func (db *FlatDB) Import(dbPath string) {
	dat, err := ioutil.ReadFile(dbPath)
	check(err)
	strs := string(dat)
	lines := strings.Split(strs, "\n")

	var prevLen uint64

	for _, line := range lines {
		// skip blank
		if len(line) == 0 {
			continue
		}

		book := csvToBook(line)

		// skip incomplete record
		if book == nil {
			continue
		}

		ibook := &IBook{
			Address: prevLen,
			Book:    book,
			Length:  uint64(len(line)),
		}

		db.IBooks = append(db.IBooks, ibook)

		db.Mapper[book.ID] = book

		prevLen += uint64(len(line) + 1)
	}
}

// Save dabase in default path
func (db *FlatDB) Save() {
	db.Export(db.Path)
}

// Export is save database to another path
func (db *FlatDB) Export(dbPath string) {
	f, err := os.Create(dbPath)
	check(err)
	defer f.Close()

	ibooks := db.IBooks
	for _, ibook := range ibooks {
		f.Write(bookToCSV(ibook.Book))
	}
}

// UpdatePage change database record on page read
func (db *FlatDB) UpdatePage(id string, page int) {
	book := db.Mapper[id]
	book.Page = uint64(page)

}

// csvToBook convert string to book
func csvToBook(line string) *Book {
	r := csv.NewReader(strings.NewReader(line))
	records, err := r.Read()
	check(err)

	// incomplete record
	if len(records) != 14 {
		return nil
	}

	book := &Book{
		ID:       records[0],
		Title:    records[11],
		Author:   records[12],
		Fullpath: records[13],
		Cond:     bookCond(records[1]),
		Pages:    mustUint64(records[2]),
		Page:     mustUint64(records[3]),
		Ranking:  mustUint64(records[4]),
		Fav:      mustUint64(records[5]),
		Size:     mustUint64(records[6]),
		Inode:    mustUint64(records[7]),
		Mtime:    mustUint64(records[8]),
		Itime:    mustUint64(records[9]),
		Rtime:    mustUint64(records[10]),
	}
	return book
}

// bookToCSV convert Book to csv bytes
func bookToCSV(book *Book) []byte {
	// DO NOT change ordering, can only append in future
	// use this a reference
	records := []string{
		book.ID,                                   //  0
		fmt.Sprint(book.Cond),                     //  1
		fmt.Sprintf(FlatDBCharsPage, book.Pages),  //  2
		fmt.Sprintf(FlatDBCharsPage, book.Page),   //  3
		fmt.Sprint(book.Ranking),                  //  4
		fmt.Sprint(book.Fav),                      //  5
		fmt.Sprintf(FlatDBCharsFSize, book.Size),  //  6
		fmt.Sprintf(FlatDBCharsEpoch, book.Inode), //  7
		fmt.Sprintf(FlatDBCharsEpoch, book.Mtime), //  8
		fmt.Sprintf(FlatDBCharsEpoch, book.Itime), //  9
		fmt.Sprintf(FlatDBCharsEpoch, book.Rtime), //  10
		book.Title,    // 11
		book.Author,   // 12
		book.Fullpath, // 13
	}

	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Write(records)
	w.Flush() // commit or data is empty

	return b.Bytes()
}

//
// debug / test code ------------------------------------------------------------------------------------------------------
//

// convJtoF makes old json db into flat db
func convJtoF() {
	in := userHome("etc/kamishibai-kai/db.json")
	out := userHome("etc/shin-kamishibai/db.txt")

	dat, err := ioutil.ReadFile(in)
	check(err)

	var result map[string]*Book
	err = json.Unmarshal(dat, &result)
	check(err)

	for id, book := range result {
		book.ID = id
	}

	f, err := os.Create(out)
	check(err)
	defer f.Close()

	for _, book := range result {
		f.Write(bookToCSV(book))
	}
}

// convFtoJ makes flat db to json
func convFtoJ() {
	in := userHome("etc/shin-kamishibai/db.txt")
	out := userHome("etc/shin-kamishibai/db.json")

	dat, err := ioutil.ReadFile(in)
	check(err)
	strs := string(dat)
	lines := strings.Split(strs, "\n")

	ibooks := []IBook{}
	var prevLen uint64

	for _, line := range lines {
		// skip blank
		if len(line) == 0 {
			continue
		}

		r := csv.NewReader(strings.NewReader(line))
		records, err := r.Read()
		check(err)

		// skip incomplete
		if len(records) != 14 {
			continue
		}

		book := &Book{
			ID:       records[0],
			Title:    records[11],
			Author:   records[12],
			Fullpath: records[13],
			Cond:     bookCond(records[1]),
			Pages:    mustUint64(records[2]),
			Page:     mustUint64(records[3]),
			Ranking:  mustUint64(records[4]),
			Fav:      mustUint64(records[5]),
			Size:     mustUint64(records[6]),
			Inode:    mustUint64(records[7]),
			Mtime:    mustUint64(records[8]),
			Itime:    mustUint64(records[9]),
			Rtime:    mustUint64(records[10]),
		}
		// fmt.Println(i, book.ID)

		ibook := &IBook{
			Address: prevLen,
			Book:    book,
			Length:  uint64(len(line)),
		}

		ibooks = append(ibooks, *ibook)

		prevLen += uint64(len(line) + 1)
	}

	jbooks := make(map[string]*Book)
	for _, ibook := range ibooks {
		// fmt.Println(ibook, ibook.Book)
		book := *ibook.Book // dereference, create clone
		book2 := &book
		book2.Cond = 0
		book2.ID = ""
		jbooks[ibook.ID] = book2
	}

	// fmt.Println(ibooks)
	jstr, err := json.MarshalIndent(jbooks, "", "  ")
	check(err)

	f, err := os.Create(out)
	defer f.Close()
	check(err)
	f.Write(jstr)

	fmt.Println(lines[100])

}
