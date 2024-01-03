package main

import (
	"log"
	"fmt"
	"sort"
	"regexp"
	"strings"
)

// String sort

func sortNatural(arr []string, filter *regexp.Regexp) []string {
	newArr := make([]string, len(arr))
	copy(newArr, arr)
	stringQuicksort(newArr, filter, 0, len(arr)-1)
	return newArr
}

func stringQuicksort(arr []string, filter *regexp.Regexp, low, high int) {
	if low < high {
		pi := stringPartition(arr, filter, low, high)
		stringQuicksort(arr, filter, low, pi-1)
		stringQuicksort(arr, filter, pi+1, high)
	}
}

func stringPartition(arr []string, filter *regexp.Regexp, low, high int) int {
	pivot := arr[high]
	i := low - 1
	for j := low; j < high; j++ {
		a := arr[j]
		b := pivot
		if filter != nil {
			a = filter.ReplaceAllString(a, "")
			b = filter.ReplaceAllString(b, "")
		}
		if a <= b {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}
	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

//
// Books filter, sort
//

func filterBooksBy(inBooks []*Book, filter, byType string) []*Book {
	books := []*Book{}

	search := strings.ToLower(filter)
	keywords := strings.Split(search, " ")
	keywords = StringSliceFlatten(keywords)

	// no filter, return books as is
	if len(keywords) == 0 {
		return inBooks
	}

	// filter supplied
OUTER:
	for _, book := range inBooks {
		foundKeywords := 0

		for _, keyword := range keywords {
			target := strings.ToLower(book.Title)
			switch byType {
			case "author":
				target = strings.ToLower(book.Author)
			case "author-title":
				target = strings.ToLower(book.Author + book.Title)
			}

			// no match, next book
			if !strings.Contains(target, keyword) {
				continue OUTER
			}

			foundKeywords++
		}

		// all keywords found, add book
		if foundKeywords >= len(keywords) {
			books = append(books, book)
		}
	}

	return books
}


func filterBooksByTitle(books []*Book, filter string) []*Book {
	return filterBooksBy(books, filter, "title")
}

func filterBooksByAuthor(books []*Book, filter string) []*Book {
	return filterBooksBy(books, filter, "author")
}

func filterBooksByAuthorTitle(books []*Book, filter string) []*Book {
	return filterBooksBy(books, filter, "author-title")
}

func sortBooksByTitle(books []*Book) []*Book {
	booksQuicksort(books, "title", 0, len(books)-1)

	return books
}

func sortBooksByAuthor(books []*Book) []*Book {
	booksQuicksort(books, "author", 0, len(books)-1)

	return books
}

func sortBooksByAuthorTitle(books []*Book) []*Book {
	// sort by author then boook title
	booksQuicksort(books, "author-title", 0, len(books)-1)

	return books
}

func sortBooksByFav(books []*Book) []*Book {
    // Custom less function to sort favorited books first
    less := func(i, j int) bool {
        return books[i].Fav > books[j].Fav
    }
    // Sort the books using the custom less function
    sort.Slice(books, less)
    return books
}


func booksQuicksort(arr []*Book, byType string, low, high int) {
	if low < high {
		var pi int = booksPartition(arr, byType, low, high)

		booksQuicksort(arr, byType, low, pi-1)
		booksQuicksort(arr, byType, pi+1, high)
	}
}

func booksPartition(arr []*Book, byType string, low, high int) int {
	var pivot *Book = arr[high]

	var i int = (low - 1)

	for j := low; j < high; j++ {
		switch byType {
		case "title":
			// sort by title + volume/chapter (cuz author could have multiple title)
			a := fmt.Sprintf("%s %s", arr[j].Title, arr[j].Number)
			b := fmt.Sprintf("%s %s", pivot.Title, pivot.Number)

			// natural compare
			if AlphaNumCaseCompare(a, b) {
				i++
				arr[i], arr[j] = arr[j], arr[i]
			}

		case "author":
			a := arr[j].Author
			b := pivot.Author

			// natural compare
			if AlphaNumCaseCompare(a, b) {
				i++
				arr[i], arr[j] = arr[j], arr[i]
			}

		case "author-title":
			a := arr[j].Author + arr[j].Title
			b := pivot.Author + pivot.Title

			// natural compare
			if AlphaNumCaseCompare(a, b) {
				i++
				arr[i], arr[j] = arr[j], arr[i]
			}

		case "fav": // not used
			// sort by title + volume/chapter (cuz author could have multiple title)
			a := arr[j].Fav
			b := pivot.Fav

			// natural compare
			if (a > b) {
				i++
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]

	return i + 1
}

//
// FileInfoBasic, support functions
//

func fibsQuicksort(arr []*FileInfoBasic, byType string, low, high int) {
	if low < high {
		var pi int = fibsPartition(arr, byType, low, high)

		fibsQuicksort(arr, byType, low, pi-1)
		fibsQuicksort(arr, byType, pi+1, high)
	}
}

func fibsPartition(arr []*FileInfoBasic, byType string, low, high int) int {
		var pivot *FileInfoBasic = arr[high]

		var i int = (low - 1)

		for j := low; j < high; j++ {
			switch byType {
		case sortOrderByFileName:
		    // sort by filename
		    a := arr[j].Name
		    b := pivot.Name
		    if a < b {
		        i++
		        arr[i], arr[j] = arr[j], arr[i]
		    }
		case sortOrderByFileModTime:
		    // sort by file modification time
		    a := arr[j].ModTime
		    b := pivot.ModTime
		    if a.After(b) { // newest first
		        i++
		        arr[i], arr[j] = arr[j], arr[i]
		    }
		case sortOrderByReadTime:
			// sort by read time
			a := arr[j].Rtime
			b := pivot.Rtime

			// natural compare
			if a < b { // oldest read first
				i++
				arr[i], arr[j] = arr[j], arr[i]
			}
		case sortOrderByAuthor:
			// sort by author and title
			a := arr[j].Book.Author + " " + arr[j].Book.Title + " " + arr[j].Book.Number
			b := pivot.Book.Author + " " + arr[j].Book.Title + " " + arr[j].Book.Number

			// natural compare
			if a > b {
				i++
				arr[i], arr[j] = arr[j], arr[i]
			}

		case sortOrderByFav: // not used
		    // Define the specific sorting logic based on the favorite status
		    // Example: Sort favorites before non-favorites
		    //if arr[j].Book.Fav > pivot.Book.Fav {
		//	i++
			arr[i], arr[j] = arr[j], arr[i]
		  //  }
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]

	return i + 1
}

//
// FileInfoBasic, sort types
//

// sort by filename
func sortByFileName(arr []*FileInfoBasic) []*FileInfoBasic {
	newArr := append([]*FileInfoBasic{}, arr...)
	fibsQuicksort(newArr, sortOrderByFileName, 0, len(arr)-1)
	return newArr
}

// sort by read time, most recent one first
func sortByReadTime(arr []*FileInfoBasic) []*FileInfoBasic {
	newArr := append([]*FileInfoBasic{}, arr...)
	fibsQuicksort(newArr, sortOrderByReadTime, 0, len(arr)-1)
	return newArr
}

// sort by file modification time
func sortByFileModTime(arr []*FileInfoBasic) []*FileInfoBasic {
	newArr := append([]*FileInfoBasic{}, arr...)
	fibsQuicksort(newArr, sortOrderByFileModTime, 0, len(arr)-1)
	return newArr
}

// sort by author by title by number
func sortByAuthorTitle(arr []*FileInfoBasic) []*FileInfoBasic {
	newArr := append([]*FileInfoBasic{}, arr...)
	fibsQuicksort(newArr, sortOrderByAuthor, 0, len(arr)-1)
	return newArr
}

// sort by fav
func sortByFav(arr []*FileInfoBasic) []*FileInfoBasic {
    newArr := append([]*FileInfoBasic{}, arr...)
    // Custom less function to sort favorited books first
    less := func(i, j int) bool {
	return newArr[i].Fav > newArr[j].Fav
    }
    // Sort the books using the custom less function
    sort.Slice(newArr, less)
    // Print the sorted books
    log.Println(arr)
    log.Println("\n\n\n")
    log.Println(newArr)
    return newArr
}
