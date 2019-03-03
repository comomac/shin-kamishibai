package main

// flat file db

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
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
	Order    string `json:"order"`          // volume, chapter
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
	IBooks      []*IBook
	Authors     []*Author
	IMapper     map[string]*IBook // map books by id
	FMapper     map[string]*IBook // map books by file path
	Path        string            // where the database is stored
	FileModDate int64             // file last modified date
}

// convert string to uint64
func mustUint64(s string) uint64 {
	i, err := strconv.Atoi(s)
	check(err)
	return uint64(i)
}

// generate random characters for the unique book ID, argument needs length
func genChar(minLen int) string {
	rand.Seed(time.Now().UnixNano())

	validChars := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") // 62 uniq chars
	var chars []byte

	ttlValidChars := len(validChars)

	for len(chars) < minLen {
		rnum := rand.Intn(ttlValidChars)

		chars = append(chars, validChars[rnum])
	}

	return string(chars)
}

// bookCond gives a numeric representation of the state of book file
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
	db.IMapper = make(map[string]*IBook)
	db.FMapper = make(map[string]*IBook)

	return db
}

// Clear all data
func (db *FlatDB) Clear() {
	db.IBooks = nil
	db.Authors = nil
	db.IMapper = make(map[string]*IBook)
	db.FMapper = make(map[string]*IBook)
}

// Load data using default file path
func (db *FlatDB) Load() {
	db.Import(db.Path)
}

// Reload data using default file path
func (db *FlatDB) Reload() {
	db.Clear()
	db.Import(db.Path)
}

// Import data from alternative path
func (db *FlatDB) Import(dbPath string) {
	// make sure db exists
	fstat, err := os.Stat(db.Path)
	check(err)

	// remember file last modified time, will use it later for checking
	db.FileModDate = fstat.ModTime().Unix()

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

		db.IMapper[book.ID] = ibook
		db.IMapper[book.Fullpath] = ibook

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

// UpdatePage change database record on page read, returns written byte size
func (db *FlatDB) UpdatePage(id string, page int) (int, error) {
	ibook := db.IMapper[id]
	if ibook == nil {
		err := errors.New("ibook is nil")
		return 0, err
	}
	ibook.Page = uint64(page)

	// read out from db
	b := make([]byte, ibook.Length)

	f, err := os.OpenFile(db.Path, os.O_RDWR, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	f.ReadAt(b, int64(ibook.Address))
	strs := string(b)

	// make sure the column spacing is still the same. ascii 44 is comma
	if strs[3] != 44 || strs[5] != 44 || strs[10] != 44 || strs[15] != 44 {
		err := errors.New("db column has changed")
		return 0, err
	}

	// page with extended chars
	strPage := fmt.Sprintf(FlatDBCharsPage, ibook.Page)
	b2 := bytes.NewBufferString(strPage)
	// absolute position for the page
	posPage := int64(ibook.Address + 11)
	b2len, err := f.WriteAt(b2.Bytes(), posPage)

	return b2len, err
}

// BookIDs gives list of all the book ids in the db
func (db *FlatDB) BookIDs() []string {
	ids := make([]string, len(db.IBooks))

	for _, ibook := range db.IBooks {
		ids = append(ids, ibook.ID)
	}

	return ids
}

// AddBook by file path, returns generated book id
func (db *FlatDB) AddBook(bookPath string) error {
	// generate unique book id
	id := genChar(3)
	// make sure book id is unique
	for db.GetBookByID(id) != nil {
		id = genChar(3)
	}

	fstat, err := os.Stat(db.Path)
	check(err)

	// get file inode
	fstat2, ok := fstat.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("Not a syscall.Stat_t")
	}

	book := &Book{
		ID: id,
		// Title:    records[11],
		// Author:   records[12],
		// Fullpath: records[13],
		Cond: bookCond(bookPath),
		// Pages: mustUint64(records[2]),
		Size:  uint64(fstat.Size()),
		Inode: fstat2.Ino,
		Mtime: uint64(fstat.ModTime().Unix()),
		Itime: uint64(time.Now().Unix()),
	}

	f, err := os.OpenFile(db.Path, os.O_APPEND, 0644)
	if err != nil {
		return errors.New("Unable to add book to db")
	}
	defer f.Close()

	f.Write(bookToCSV(book))

	db.Reload()

	return nil
}

func getAuthor(str string) string {
	// get first [...]
	result := regexp.MustCompile(`\[(.+?)\]`).FindStringSubmatch(str)
	if result != nil {
		return result[1]
	}

	// no author found
	return ""
}

func getTitle(str string) string {
	s := str
	// get rid of extension, case insensitive
	s = regexp.MustCompile(`(?i).cbz`).ReplaceAllString(s, ``)
	// get rid of english
	s = regexp.MustCompile(` - [ \?\!\-\+\.\~\(\)\[\]A-Za-z0-9]+`).ReplaceAllString(s, ``)
	// underline to space
	s = regexp.MustCompile(`_`).ReplaceAllString(s, ` `)
	// change unicode wide space to narrow(ascii) space
	s = regexp.MustCompile(`　`).ReplaceAllString(s, ` `)
	// get rid of (...)
	s = regexp.MustCompile(`\(.+?\)`).ReplaceAllString(s, ``)
	// get rid of [...]
	s = regexp.MustCompile(`\[.+?\]`).ReplaceAllString(s, ``)
	//s.gsub!(/ \S\d+.*/,'')

	// get rid of vol or chapter
	s = strings.Replace(s, getVolumeOrChapter(str), ``, -1)

	// change multi-spaces to single space
	s = regexp.MustCompile(` +`).ReplaceAllString(s, ` `)
	// trim leading and trailing space
	s = strings.TrimSpace(s)

	return s
}

func getVolumeOrChapter(str string) string {
	var result []string

	// remove extension
	s := regexp.MustCompile(`\.cbz`).ReplaceAllString(str, ``)

	// change unicode wide space to narrow(ascii) space
	s = regexp.MustCompile(`　`).ReplaceAllString(s, ` `)

	// e.g.  第01巻   第1-3話
	result = regexp.MustCompile(`(?i) (第[\d\-\~]+)(巻|部|話)`).FindStringSubmatch(s)
	if result != nil {
		return result[1] + result[2]
	}

	// e.g.  1巻   1-3話
	result = regexp.MustCompile(`(?i) ([\d\-\~]+)(巻|部|話)`).FindStringSubmatch(s)
	if result != nil {
		return result[1] + result[2]
	}

	// e.g.  上巻
	result = regexp.MustCompile(`(?i) (上|中|下)(巻|部|話)`).FindStringSubmatch(s)
	if result != nil {
		return result[1] + result[2]
	}

	// e.g.  vol.01.cbz
	result = regexp.MustCompile(`(?i) (vol.{0,2}\d+)$`).FindStringSubmatch(s)
	if result != nil {
		return result[1]
	}

	// e.g.  ch.01.cbz
	result = regexp.MustCompile(`(?i) (ch.{0,2}\d+)$`).FindStringSubmatch(s)
	if result != nil {
		return result[1]
	}

	// e.g.  上.cbz
	result = regexp.MustCompile(`(?i) (上|中|下)$`).FindStringSubmatch(s)
	if result != nil {
		return result[1]
	}

	// e.g.  v01.cbz
	result = regexp.MustCompile(`(?i) (v\d+\.)$`).FindStringSubmatch(s)
	if result != nil {
		return result[1]
	}

	// e.g.  c01.cbz
	result = regexp.MustCompile(`(?i) (c\d+\.)$`).FindStringSubmatch(s)
	if result != nil {
		return result[1]
	}

	// 2018年10月号
	result = regexp.MustCompile(`(?i) (\d{4}年\d{2}月号)`).FindStringSubmatch(s)
	if result != nil {
		return result[1]
	}

	// commented out. because some book title could have number
	// // e.g.  01.cbz
	// result = regexp.MustCompile(`(?i) (\d+)$`).FindStringSubmatch(s)
	// if result != nil {
	// 	return result[1]
	// }

	return ""
}

// GetBookByID get Book object by book id
func (db *FlatDB) GetBookByID(bookID string) *Book {

	return db.IMapper[bookID].Book
}

// GetBookByPath get Book object by file path
func (db *FlatDB) GetBookByPath(fpath string) *Book {

	return db.FMapper[fpath].Book
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
	fname := path.Base(book.Fullpath) // delete me

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
		getTitle(fname),                           // book.Title,    // 11
		getAuthor(fname),                          // book.Author,   // 12
		getVolumeOrChapter(fname),                 // TODO delete me
		book.Fullpath,                             // 13
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
