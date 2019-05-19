package main

import (
	"os/user"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func userHome(s ...string) string {
	user, err := user.Current()
	check(err)

	dir := user.HomeDir

	if len(s) == 0 {
		return dir
	}

	if []rune(s[0])[0] != '/' {
		return dir + "/" + s[0]
	}

	return dir + s[0]
}

// BookInfoBasic contains basic book information
type BookInfoBasic struct {
	Title  string
	Author string
	Volume int
	Images int
}

func getBookInfoBasic(filepath string) {

}

// StringSliceContain search the string slice and see if it contains the searched word, match from first character
func StringSliceContain(strSlice []string, strSearch string) bool {
	for _, str := range strSlice {
		if strings.Index(strSearch, str) == 0 {
			return true
		}
	}

	return false
}
