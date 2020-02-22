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
	stringQuicksort(arr, filter, 0, len(arr)-1)

	return arr
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

	for j := low; j < high; j++ {
		a := &arr[j]
		b := pivot

		// deref for pure string, and for manipulation
		a2 := *a
		b2 := *b
		if filter != nil {
			// filtered
			a2 = filter.ReplaceAllString(a2, "")
			b2 = filter.ReplaceAllString(b2, "")
		}

		// natural compare
		if AlphaNumCaseCompare(a2, b2) {
			i++

			var temp *string = &arr[i]
			arr[i] = arr[j]
			arr[j] = *temp
		}
	}

	var temp1 *string = &arr[i+1]
	arr[i+1] = arr[high]
	arr[high] = *temp1

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
	booksQuicksort(books, "title", 0, len(books))

	return books
}

func sortByAuthor(books []*Book) []*Book {
	booksQuicksort(books, "author", 0, len(books))

	return books
}

func sortByAuthorTitle(books []*Book) []*Book {
	// sort by author then boook title
	booksQuicksort(books, "author-title", 0, len(books))

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
			a := fmt.Sprintf("%s %s", arr[i].Title, arr[i].Number)
			b := fmt.Sprintf("%s %s", pivot.Title, pivot.Number)

			// natural compare
			if AlphaNumCaseCompare(a, b) {
				i++

				var temp *Book = arr[i]
				arr[i] = arr[j]
				arr[j] = temp
			}

		case "author":
			a := arr[j].Author
			b := pivot.Author

			// natural compare
			if AlphaNumCaseCompare(a, b) {
				i++

				var temp *Book = arr[i]
				arr[i] = arr[j]
				arr[j] = temp
			}

		case "author-title":
			a := arr[j].Author + arr[j].Title
			b := pivot.Author + pivot.Title

			// natural compare
			if AlphaNumCaseCompare(a, b) {
				i++

				var temp *Book = arr[i]
				arr[i] = arr[j]
				arr[j] = temp
			}
		}
	}

	var temp1 *Book = arr[i+1]
	arr[i+1] = arr[high]
	arr[high] = temp1

	return i + 1
}
