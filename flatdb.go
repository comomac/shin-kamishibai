package main

// flat file db

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FlatDBCharsPage is number of characters reserved for the pages/page
const FlatDBCharsPage = "%04d"

// FlatDBCharsFSize is number of character reserved for the epoch time
const FlatDBCharsFSize = "%010d"

// FlatDBCharsEpoch is number of character reserved for the epoch time
const FlatDBCharsEpoch = "%010d"

// RegexSupportedImageExt supported image extension
var RegexSupportedImageExt = regexp.MustCompile(`(?i)\.(jpg|jpeg|gif|png)$`)

// Book contains all the information of book
type Book struct {
	ID       string `json:"id,omitempty"`   // unique id for indexing
	Title    string `json:"title"`          // book title
	Author   string `json:"author"`         // book author, seperated by comma
	Number   string `json:"number"`         // volume, chapter, etc
	Fullpath string `json:"-"`              // book file path
	Ranking  uint64 `json:"ranking"`        // 1-5 ranking, least to most liked
	Fav      uint64 `json:"fav"`            // favourite, 0 false, 1 true
	Cond     uint64 `json:"cond,omitempty"` // 0 unknown, 1 exists, 2 not exist, 3 deleted, 4 inaccessible
	Pages    uint64 `json:"pages"`          // total pages
	Page     uint64 `json:"page"`           // read upto
	Size     uint64 `json:"size"`           // fs file size
	Inode    uint64 `json:"-"`              // fs inode
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
	mutex        *sync.Mutex
	books        []*Book
	ibooks       []*IBook
	authors      []*Author
	mapperID     map[string]*IBook   // map books by id (unique)
	mapperPath   map[string]*IBook   // map books by file path (unique)
	mapperTitle  map[string][]*IBook // group books by title (array)
	mapperAuthor map[string][]*IBook // group books by author (array)
	Path         string              // where the database is stored
	FileModDate  int64               // file last modified date
}

// convert string to uint64
func mustUint64(s string) uint64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
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

// New initialize new Flat Database
func (db *FlatDB) New(dbPath string) {
	db.mutex = &sync.Mutex{}
	db.Path = dbPath
	db.mapperID = make(map[string]*IBook)
	db.mapperPath = make(map[string]*IBook)
	db.mapperTitle = make(map[string][]*IBook)
	db.mapperAuthor = make(map[string][]*IBook)
}

// Clear all data
func (db *FlatDB) Clear() {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.ibooks = nil
	db.authors = nil
	db.mapperID = make(map[string]*IBook)
	db.mapperPath = make(map[string]*IBook)
	db.mapperTitle = make(map[string][]*IBook)
	db.mapperAuthor = make(map[string][]*IBook)
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
func (db *FlatDB) Import(dbPath string) error {
	// make sure db exists
	fstat, err := os.Stat(db.Path)
	if os.IsNotExist(err) {
		// create blank not exist
		f, err := os.OpenFile(db.Path, os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
	}
	if err != nil {
		return err
	}

	// remember file last modified time, will use it later for checking
	db.FileModDate = fstat.ModTime().Unix()

	dat, err := ioutil.ReadFile(dbPath)
	if err != nil {
		return err
	}
	strs := string(dat)
	lines := strings.Split(strs, "\n")

	var prevLen uint64

	for _, line := range lines {
		// skip blank
		if len(line) == 0 {
			continue
		}

		book, _ := csvToBook(line)
		// skip incomplete record
		if book == nil {
			continue
		}

		ibook := &IBook{
			Address: prevLen,
			Book:    book,
			Length:  uint64(len(line)),
		}

		db.mutex.Lock()
		db.books = append(db.books, book)
		db.ibooks = append(db.ibooks, ibook)
		db.mapperID[book.ID] = ibook
		db.mapperPath[book.Fullpath] = ibook
		db.mapperTitle[book.Title] = append(db.mapperTitle[book.Title], ibook)
		db.mapperAuthor[book.Author] = append(db.mapperAuthor[book.Author], ibook)
		db.mutex.Unlock()

		prevLen += uint64(len(line) + 1)
	}

	return nil
}

// Save dabase in default path
func (db *FlatDB) Save() {
	db.Export(db.Path)
}

// Export is save database to another path
func (db *FlatDB) Export(dbPath string) error {
	f, err := os.Create(dbPath)
	if err != nil {
		return err
	}
	defer f.Close()

	ibooks := db.ibooks
	for _, ibook := range ibooks {
		f.Write(bookToCSV(ibook.Book))
	}
	return nil
}

// UpdatePage change database record on page read, returns written byte size
func (db *FlatDB) UpdatePage(id string, page int) (int, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	ibook := db.mapperID[id]
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
	ids := make([]string, len(db.ibooks))

	for _, ibook := range db.ibooks {
		ids = append(ids, ibook.ID)
	}

	return ids
}

// AddBook by file path, returns generated book id. return non-pointer book because db will be reloaded
func (db *FlatDB) AddBook(bookPath string) (*Book, error) {
	// generate unique book id
	id := genChar(3)
	// make sure book id is unique
	for db.GetBookByID(id) != nil {
		id = genChar(3)
	}

	fstat, err := os.Stat(bookPath)
	if err != nil {
		return nil, err
	}

	// // get file inode
	// fstat2, ok := fstat.Sys().(*syscall.Stat_t)
	// if !ok {
	// 	return nil, errors.New("Not a syscall.Stat_t")
	// }

	pages, err := cbzGetPages(bookPath)
	if err != nil {
		fmt.Println("error! failed to add book", bookPath, err)
		return nil, err
	}

	// filename
	fname := path.Base(bookPath)

	book := Book{
		ID:       id,
		Title:    getTitle(fname),
		Author:   getAuthor(fname),
		Number:   getNumber(fname),
		Fullpath: bookPath,
		Cond:     bookCond(bookPath),
		Pages:    uint64(pages),
		Size:     uint64(fstat.Size()),
		// Inode:    fstat2.Ino,
		Mtime: uint64(fstat.ModTime().Unix()),
		Itime: uint64(time.Now().Unix()),
	}

	f, err := os.OpenFile(db.Path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// save to db file
	b := bookToCSV(&book)
	f.Write(b)
	f.Close()

	// reload from db file
	db.Reload()

	return &book, nil
}

func visit(db *FlatDB) func(string, os.FileInfo, error) error {
	return func(fpath string, f os.FileInfo, err error) error {
		// skip folder
		if f.IsDir() {
			return nil
		}
		// skip dot file
		if strings.HasPrefix(fpath, ".") {
			return nil
		}
		if strings.HasPrefix(f.Name(), ".") {
			return nil
		}
		// skip non cbz extension
		fpath2 := strings.ToLower(fpath)
		if !strings.HasSuffix(fpath2, ".cbz") {
			return nil
		}

		// get file state, e.g. size
		fstat, err := os.Stat(fpath)
		if err != nil {
			return err
		}

		// make sure books are unique so no duplicate db record
		fname := path.Base(fpath)
		books := db.SearchBookByNameAndSize(fname, uint64(fstat.Size()))
		if len(books) > 0 {
			// skip
			return nil
		}

		_, err = db.AddBook(fpath)
		if err != nil {
			return err
		}
		fmt.Println("Added book", fpath)

		return nil
	}
}

// AddFile adds book to db
func (db *FlatDB) AddFile(fpath string) (*Book, error) {
	var err error

	// get file state, e.g. size
	f, err := os.Stat(fpath)
	if err != nil {
		return nil, err
	}

	// skip folder
	if f.IsDir() {
		return nil, errors.New("not a file")
	}
	// skip dot file
	if strings.HasPrefix(f.Name(), ".") {
		return nil, errors.New("no dot file")
	}
	// skip non cbz extension
	lname := strings.ToLower(f.Name())
	if !strings.HasSuffix(lname, ".cbz") {
		return nil, errors.New("not a cbz")
	}

	// make sure books are unique so no duplicate db record
	books := db.SearchBookByNameAndSize(f.Name(), uint64(f.Size()))
	if len(books) > 0 {
		// skip
		return nil, errors.New("dup book found")
	}

	book, err := db.AddBook(fpath)
	if err != nil {
		return nil, err
	}
	fmt.Println("Added book", fpath)

	return book, nil
}

// AddDirR recursively add books from directory
func (db *FlatDB) AddDirR(dir string) error {
	return filepath.Walk(dir, visit(db))
}

// AddDir add books from directory
func (db *FlatDB) AddDir(dir string) error {
	// filepath.Glob() dont work with unicode file name dir so using ioutil.ReadDir()
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = visit(db)(filepath.Join(dir, file.Name()), file, err)
		if err != nil {
			return err
		}
	}

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
	s = regexp.MustCompile(` - [ \?\!\-\+\.\,\~\(\)\[\]A-Za-z0-9]+`).ReplaceAllString(s, ``)
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
	s = strings.Replace(s, getNumber(str), ``, -1)

	// change multi-spaces to single space
	s = regexp.MustCompile(` +`).ReplaceAllString(s, ` `)
	// trim leading and trailing space
	s = strings.TrimSpace(s)

	return s
}

func getNumber(str string) string {
	var result []string

	// remove extension
	s := regexp.MustCompile(`\.cbz`).ReplaceAllString(str, ``)

	// change unicode wide space to narrow(ascii) space
	s = regexp.MustCompile(`　`).ReplaceAllString(s, ` `)

	// e.g.  第01巻   第1-3話     巻(まき)  卷(かん)
	result = regexp.MustCompile(`(?i) (第[\d\-\~]+)(巻|卷|部|話)`).FindStringSubmatch(s)
	if result != nil {
		return result[1] + result[2]
	}

	// e.g.  1巻   1-3話
	result = regexp.MustCompile(`(?i) ([\d\-\~\,]+)(巻|卷|部|話)`).FindStringSubmatch(s)
	if result != nil {
		return result[1] + result[2]
	}

	// e.g.  上巻
	result = regexp.MustCompile(`(?i) (上|中|下)(巻|卷|部|話)`).FindStringSubmatch(s)
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
	result = regexp.MustCompile(`(?i) (v\d+)$`).FindStringSubmatch(s)
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

// cbzGetPages find out how many pages in cbz
func cbzGetPages(fp string) (int, error) {
	zr, err := zip.OpenReader(fp)
	if err != nil {
		return -1, err
	}
	defer zr.Close()

	i := 0
	for _, f := range zr.File {
		if RegexSupportedImageExt.MatchString(f.Name) {
			i++
		}
	}

	if i == 0 {
		return -1, errors.New("not a cbz")
	}

	return i, nil
}

// GetBookByID get Book object by book id
func (db *FlatDB) GetBookByID(bookID string) *Book {

	db.mutex.Lock()
	defer db.mutex.Unlock()

	ibook := db.mapperID[bookID]
	if ibook == nil {
		return nil
	}

	return ibook.Book
}

// GetBookByPath get Book object by file path
func (db *FlatDB) GetBookByPath(fpath string) *Book {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// not working for some reason
	ibook := db.mapperPath[fpath]
	if ibook == nil {
		return nil
	}
	return db.mapperPath[fpath].Book
}

// SearchBookByNameAndSize get Books object by filename and size
func (db *FlatDB) SearchBookByNameAndSize(fname string, size uint64) []*Book {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	var books []*Book

	for _, ibook := range db.ibooks {
		if ibook.Size == size && path.Base(ibook.Fullpath) == fname {
			books = append(books, ibook.Book)
		}
	}

	return books
}

// Search find Books base on title and author
func (db *FlatDB) Search(search string) []*Book {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	books := filterByAuthorTitle(db.books, search)

	return books
}

//
// helper code ------------------------------------------------------------------------------------------------------
//

// csvToBook convert string to book
func csvToBook(line string) (*Book, error) {
	r := csv.NewReader(strings.NewReader(line))
	records, err := r.Read()
	if err != nil {
		return nil, err
	}

	// incomplete record
	if len(records) != 15 {
		return nil, errors.New("incomplete line")
	}

	book := &Book{
		ID:       records[0],
		Title:    records[11],
		Author:   records[12],
		Number:   records[13],
		Fullpath: records[14],
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

	return book, nil
}

// bookToCSV convert Book to csv bytes
func bookToCSV(book *Book) []byte {
	// book file name
	fname := path.Base(book.Fullpath)

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
		fmt.Sprintf(FlatDBCharsEpoch, book.Rtime), // 10
		getTitle(fname),                           // book.Title,   // 11
		getAuthor(fname),                          // book.Author,  // 12
		getNumber(fname),                          // book.Number,  // 13
		book.Fullpath,                             // 14
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

// ConvJtoF makes old json db into flat db
func ConvJtoF(in, out string) error {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}

	var result map[string]*Book
	err = json.Unmarshal(dat, &result)
	if err != nil {
		return err
	}

	for id, book := range result {
		book.ID = id
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, book := range result {
		f.Write(bookToCSV(book))
	}

	return nil
}

// ConvFtoJ makes flat db to json
func ConvFtoJ(in, out string) error {
	dat, err := ioutil.ReadFile(in)
	if err != nil {
		return err
	}
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
		if err != nil {
			return err
		}

		// skip incomplete
		if len(records) != 14 {
			continue
		}

		book := &Book{
			ID:       records[0],
			Title:    records[11],
			Author:   records[12],
			Number:   records[13],
			Fullpath: records[14],
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
	if err != nil {
		return err
	}

	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write(jstr)

	fmt.Println(lines[100])

	return nil
}
