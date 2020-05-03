package main

import (
	"fmt"
	"regexp"
	"strings"
)

//
// String sort
//

func sortNatural(arr []string, filter *regexp.Regexp) []string {
	newArr := append([]string{}, arr...)
	// free memory
	defer func() {
		newArr = nil
	}()

	stringQuicksort(newArr, filter, 0, len(arr)-1)

	return newArr
}

func stringQuicksort(arr []string, filter *regexp.Regexp, low, high int) {
	if low < high {
		var pi int = stringPartition(arr, filter, low, high)

		stringQuicksort(arr, filter, low, pi-1)
		stringQuicksort(arr, filter, pi+1, high)
	}
}

func stringPartition(arr []string, filter *regexp.Regexp, low, high int) int {
	var pivot *string = &arr[high]

	var i int = (low - 1)

	for j := low; j <= high-1; j++ {
		// deref for pure string, and for manipulation
		a := arr[j]
		b := *pivot
		if filter != nil {
			// filtered
			a = filter.ReplaceAllString(a, "")
			b = filter.ReplaceAllString(b, "")
		}

		// natural compare
		if AlphaNumCaseCompare(a, b) {
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

func filterBy(inBooks []*Book, filter, byType string) []*Book {
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

func filterByTitle(books []*Book, filter string) []*Book {
	return filterBy(books, filter, "title")
}

func filterByAuthor(books []*Book, filter string) []*Book {
	return filterBy(books, filter, "author")
}

func filterByAuthorTitle(books []*Book, filter string) []*Book {
	return filterBy(books, filter, "author-title")
}

func sortByTitle(books []*Book) []*Book {
	booksQuicksort(books, "title", 0, len(books)-1)

	return books
}

func sortByAuthor(books []*Book) []*Book {
	booksQuicksort(books, "author", 0, len(books)-1)

	return books
}

func sortByAuthorTitle(books []*Book) []*Book {
	// sort by author then boook title
	booksQuicksort(books, "author-title", 0, len(books)-1)

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
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]

	return i + 1
}

//
// FileInfoBasic, sort
//

// sort by filename
func sortByFileName(arr []*FileInfoBasic) []*FileInfoBasic {
	newArr := append([]*FileInfoBasic{}, arr...)
	// free memory
	defer func() {
		newArr = nil
	}()

	fibsQuicksort(newArr, "name", 0, len(arr)-1)

	return newArr
}

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
		case "name":
			// sort by filename
			a := fmt.Sprintf("%s", arr[j].Name)
			b := fmt.Sprintf("%s", pivot.Name)

			// natural compare
			if AlphaNumCaseCompare(a, b) {
				i++
				arr[i], arr[j] = arr[j], arr[i]
			}
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]

	return i + 1
}
