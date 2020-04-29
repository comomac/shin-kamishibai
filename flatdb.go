package main

// flat file db

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// FlatDBCharsPage is number of characters reserved for the pages/page
const FlatDBCharsPage = "%04d"

// FlatDBCharsFSize is number of character reserved for the epoch time
const FlatDBCharsFSize = "%010d"

// FlatDBCharsEpoch is number of character reserved for the epoch time
const FlatDBCharsEpoch = "%010d"

// RegexSupportedImageExt supported image extension
var RegexSupportedImageExt = regexp.MustCompile(`(?i)\.(jpg|jpeg|gif|png)$`)

// errors for flatdb
var (
	ErrNoBookID        = errors.New("no such book id")
	ErrNotFile         = errors.New("not a file")
	ErrDotFile         = errors.New("no dot file")
	ErrNotBook         = errors.New("not a book file")
	ErrDupBook         = errors.New("dup book")
	ErrNilIBook        = errors.New("ibook is nil")
	ErrDBColumnChanged = errors.New("db column has changed")
	ErrCSVIncomplete   = errors.New("incomplete csv line")
)

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

// aid debugging
func (b Book) String() string {
	return fmt.Sprintf(
		`{ID:%s Title:%q Author:%q Number:%q Ranking:%d Fav:%d Cond:%d Pages:%d Page:%d Size:%d Inode:%d Mtime:%d Itime:%d Rtime:%d Fullpath:%q }`,
		b.ID,
		b.Title,
		b.Author,
		b.Number,
		b.Ranking,
		b.Fav,
		b.Cond,
		b.Pages,
		b.Page,
		b.Size,
		b.Inode,
		b.Mtime,
		b.Itime,
		b.Rtime,
		b.Fullpath)
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
	mapperID     map[string]*Book   // map books by id (unique)
	mapperIID    map[string]*IBook  // map ibooks by id (unique)
	mapperPath   map[string]*Book   // map books by file path (unique)
	mapperTitle  map[string][]*Book // group books by title (array)
	mapperAuthor map[string][]*Book // group books by author (array)
	Path         string             // where the database is stored
	FileModDate  int64              // file last modified date
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
	db.mapperID = make(map[string]*Book)
	db.mapperIID = make(map[string]*IBook)
	db.mapperPath = make(map[string]*Book)
	db.mapperTitle = make(map[string][]*Book)
	db.mapperAuthor = make(map[string][]*Book)
}

// Clear all data
func (db *FlatDB) Clear() {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.books = nil
	db.ibooks = nil
	db.authors = nil
	db.mapperID = make(map[string]*Book)
	db.mapperIID = make(map[string]*IBook)
	db.mapperPath = make(map[string]*Book)
	db.mapperTitle = make(map[string][]*Book)
	db.mapperAuthor = make(map[string][]*Book)
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
	fmt.Println("importing...")
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
			prevLen += uint64(len(line) + 1)
			continue
		}
		// skip leading #
		if line[0:1] == "#" {
			prevLen += uint64(len(line) + 1)
			continue
		}

		// parse csv line
		book, err := csvToBook(line)
		if err != nil {
			prevLen += uint64(len(line) + 1)
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
		db.mapperID[book.ID] = book
		db.mapperIID[book.ID] = ibook
		db.mapperPath[book.Fullpath] = book
		db.mapperTitle[book.Title] = append(db.mapperTitle[book.Title], book)
		db.mapperAuthor[book.Author] = append(db.mapperAuthor[book.Author], book)
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

	ibook := db.mapperIID[id]
	if ibook == nil {
		return 0, ErrNilIBook
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
		return 0, ErrDBColumnChanged
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
		if strings.HasPrefix(f.Name(), ".") {
			return nil
		}

		// add book, with sanity checks
		db.AddFile(fpath)

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
		return nil, ErrNotFile
	}
	// skip dot file
	if strings.HasPrefix(f.Name(), ".") {
		return nil, ErrDotFile
	}
	// skip non cbz extension
	lname := strings.ToLower(f.Name())
	if !strings.HasSuffix(lname, ".cbz") {
		return nil, ErrNotBook
	}

	// make sure books are unique so no duplicate db record
	book := db.GetBookByPath(fpath)
	if book != nil {
		// skip
		return nil, ErrDupBook
	}

	book, err = db.AddBook(fpath)
	if err != nil {
		return nil, err
	}
	log.Println("Added book", fpath)

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

	// force free memory with GC
	zr = nil

	if i == 0 {
		return -1, ErrNotBook
	}

	return i, nil
}

// GetBookByID get Book object by book id
func (db *FlatDB) GetBookByID(bookID string) *Book {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	return db.mapperID[bookID]
}

// GetBookByPath get Book object by file path
func (db *FlatDB) GetBookByPath(fpath string) *Book {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	return db.mapperPath[fpath]
}

// GetPageCoverByID get book cover page
func (db *FlatDB) GetPageCoverByID(bookID string) ([]byte, error) {
	book := db.GetBookByID(bookID)
	if book == nil {
		return nil, ErrNotBook
	}

	zr, err := zip.OpenReader(book.Fullpath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	// get zip file list
	files := []string{}
	for _, f := range zr.File {
		if !RegexSupportedImageExt.MatchString(f.Name) {
			continue
		}

		files = append(files, f.Name)
	}

	// do natural sort
	files = sortNatural(files, RegexSupportedImageExt)

	// get first image file
	var rc io.ReadCloser
	for _, f := range zr.File {
		if f.Name != files[0] {
			continue
		}

		// get image data
		rc, err = f.Open()
		if err != nil {
			rc.Close()
			return nil, err
		}
		defer rc.Close()
		break
	}

	// generate thumb
	imgDat, err := ImageThumb(rc)
	if err != nil {
		return nil, err
	}
	if len(imgDat) == 0 {
		return nil, errors.New("image size is zero")
	}

	return imgDat, nil
}

// SearchBookByNameAndSize get Books object by filename and size
func (db *FlatDB) SearchBookByNameAndSize(fname string, size uint64) []*Book {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	var books []*Book

	for _, book := range db.books {
		if book.Size == size && path.Base(book.Fullpath) == fname {
			books = append(books, book)
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
	chrs := explode(line, -1)

	records := []string{}
	innerC := false
	innerStr := ""
	for i, chr := range chrs {
		// add quote
		if chr == "\"" && i < len(chrs) && chrs[i-1] == "\"" {
			continue
		}
		if chr == "\"" && i < len(chrs)-1 && chrs[i+1] == "\"" {
			innerStr += "\""
			continue
		}
		// flip between quotes
		if chr == "\"" {
			innerC = !innerC
			continue
		}

		if !innerC {
			if chr != "," {
				innerStr += chr
				continue
			}

			if chr == "," {
				// seperator, record buffer and start next column
				records = append(records, innerStr)
				innerStr = ""
				continue
			}
		}

		innerStr += chr
	}
	// add last column
	records = append(records, innerStr)

	// incomplete record
	if len(records) != 15 {
		return nil, ErrCSVIncomplete
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

// copied from strings.explode
// explode splits s into a slice of UTF-8 strings,
// one string per Unicode character up to a maximum of n (n < 0 means no limit).
// Invalid UTF-8 sequences become correct encodings of U+FFFD.
func explode(s string, n int) []string {
	l := utf8.RuneCountInString(s)
	if n < 0 || n > l {
		n = l
	}
	a := make([]string, n)
	for i := 0; i < n-1; i++ {
		ch, size := utf8.DecodeRuneInString(s)
		a[i] = s[:size]
		s = s[size:]
		if ch == utf8.RuneError {
			a[i] = string(utf8.RuneError)
		}
	}
	if n > 0 {
		a[n-1] = s
	}
	return a
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

	result := []string{}
	for _, str := range records {
		result = append(result, stringToCSVSafe(str))
	}
	strResult := strings.Join(result, ",") + "\n"

	return []byte(strResult)
}

// stringToCSVSafe convert string to csv safe string
func stringToCSVSafe(str string) string {
	if strings.Index(str, ",") == -1 && strings.Index(str, "\"") == -1 {
		return str
	}

	// escape "
	str2 := strings.Replace(str, "\"", "\"\"", -1)

	return "\"" + str2 + "\""
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

	books := []*Book{}
	var prevLen uint64

	for _, line := range lines {
		// skip blank
		if len(line) == 0 {
			prevLen += uint64(len(line) + 1)
			continue
		}
		// skip leading #
		if line[0:1] == "#" {
			prevLen += uint64(len(line) + 1)
			continue
		}

		// parse csv line
		book, err := csvToBook(line)
		if err != nil {
			prevLen += uint64(len(line) + 1)
			continue
		}

		books = append(books, book)

		prevLen += uint64(len(line) + 1)
	}

	jbooks := make(map[string]*Book)
	for _, book := range books {
		book2 := *book // dereference, make clone
		book2.Cond = 0
		book2.ID = ""
		jbooks[book2.ID] = &book2
	}

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

	return nil
}
