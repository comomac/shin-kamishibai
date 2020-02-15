package main

// this generate binfile.go

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BinFile contains individual file data and meta-data
type BinFile struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
	Sys     interface{}
	Data    []byte // used during program runtime
	Data64  string // used during generate
}

// lazy check, exit if error
func check(err error) {
	if err != nil {
		fmt.Println("### exited with failer!")
		panic(err)
	}
}

const binfileTemplate = `&BinFile{
	Name:    "{{.Name}}",
	Size:    {{.Size}},
	Mode:    0644,
	ModTime: time.Unix({{.ModTime.Unix}}, 0),
	IsDir:   {{.IsDir}},
}

`

func main() {
	fmt.Println("starting...")

	var err error

	// 	b1 := []byte(`package main`)
	// 	b2 := []byte(`
	// import "fmt"
	// func main() {
	//   fmt.Println("hello")
	// }`)

	// 	b := append(b1, b2...)

	// 	err := ioutil.WriteFile("hello.go", b, 0644)
	// 	if err != nil {
	// 		fmt.Println("failed to write file", err)
	// 		return
	// 	}

	// txt := []byte("the quick brown fox")
	// bs := base64.StdEncoding.EncodeToString(txt)
	// fmt.Println(bs)

	fptr, err := os.OpenFile("binfile.go", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	check(err)
	fptr.WriteString(`package main

import (
	"os"
	"time"
)

// BinFile is structure of file in source
type BinFile struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
	Sys     interface{}
	Data    []byte // used during program runtime
	Data64  string // used during generate
}

`)

	// fptr.Write([]byte(bs))
	// fptr.Close()
	// return

	allowedExt := []string{"jpg", "jpeg", "png", "gif", "htm", "html", "css", "js"}

	// start blank
	binMap := map[string]int{}

	// file counter
	i := 0

	err = filepath.Walk("web", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// no dir
		if info.IsDir() {
			return nil
		}
		// no . file
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		// only allowed file ext
		found := false
		for _, ext := range allowedExt {
			if strings.HasSuffix(strings.ToLower(info.Name()), "."+ext) {
				found = true
				break
			}
		}
		if !found {
			return nil
		}

		i++
		binMap[path] = i

		bf := BinFile{
			Name:    info.Name(),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		fptr.WriteString(fmt.Sprintf("var __binfile%d = ", i))

		td, terr := template.New(fmt.Sprintf("data%d", i)).Parse(binfileTemplate)
		tmpl := template.Must(td, terr)
		err = tmpl.Execute(fptr, bf)
		check(err)

		fmt.Println(i, bf)

		return nil
	})
	check(err)

	binmapTemplate := `var __binmapName = map[string]*BinFile{
{{range $k, $v := .}}{{printf "\t"}}"{{$k}}": __binfile{{$v}},
{{end}}`

	td, terr := template.New(fmt.Sprintf("map%d", i)).Parse(binmapTemplate)
	tmpl := template.Must(td, terr)
	err = tmpl.Execute(fptr, binMap)
	check(err)

	fptr.WriteString("}")
	fptr.Close()

	fmt.Println("done")
}
