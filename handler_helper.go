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

	filter = strings.ToLower(filter)
	keywords := strings.Split(filter, " ")

	regexs := []*regexp.Regexp{}
	for _, keyword := range keywords {
		re := regexp.MustCompile("(?i)" + strings.TrimSpace(keyword))
		regexs = append(regexs, re)
	}

OUTER:
	for _, book := range inBooks {
		// just add if no filter
		if len(regexs) == 0 {
			books = append(books, book)
			continue OUTER
		}

	INNER:
		// if there is filter, make sure target matches
		for _, re := range regexs {
			target := ""
			switch byType {
			case "title":
				target = strings.ToLower(book.Title)
			case "author":
				target = strings.ToLower(book.Author)
			}

			if re.FindStringIndex(target) != nil {
				books = append(books, book)
				continue OUTER
			} else {
				continue INNER
			}
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

func sortByTitle(books []*Book) []*Book {
	booksQuicksort(books, "title", 0, len(books))

	return books
}

func sortByAuthor(books []*Book) []*Book {
	booksQuicksort(books, "author", 0, len(books))

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
		}
	}

	var temp1 *Book = arr[i+1]
	arr[i+1] = arr[high]
	arr[high] = temp1

	return i + 1
}
