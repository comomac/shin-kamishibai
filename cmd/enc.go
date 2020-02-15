package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
)

func main() {
	fmt.Println("encode...")

	b, err := ioutil.ReadFile("web/smile.jpg")
	if err != nil {
		panic(err)
	}
	bs := base64.URLEncoding.EncodeToString(b)
	fmt.Println(bs)
}
