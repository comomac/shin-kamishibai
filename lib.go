package main

import "os/user"

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
