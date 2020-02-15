package main

// this generate binfile.go

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// lazy check, exit if error
func check(err error) {
	if err != nil {
		panic(err)
	}
}

const binfileTemplate = `&BinFile{
	Name:    "{{.Name}}",
	Size:    {{.Size}},
	Mode:    0644,
	ModTime: time.Unix({{.ModTime.Unix}}, 0),
	IsDir:   {{.IsDir}},
	`

const binmapTemplate = `var __binmapName = map[string]*BinFile{
{{range $k, $v := .}}{{printf "\t"}}"{{$k}}": __binfile{{$v}},
{{end}}`

func main() {
	fmt.Println("starting...")

	var err error

	tmplStr, err := ioutil.ReadFile("cmd/gen/template.go")
	if err != nil {
		panic("failed to load template")
	}

	fptr, err := os.OpenFile("binfile.go", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	check(err)
	fptr.Write(tmplStr)
	fptr.WriteString("\n")

	allowedExt := []string{"jpg", "jpeg", "png", "gif", "htm", "html", "css", "js"}

	// start blank
	binMap := map[string]int{}

	// file counter
	i := 0

	err = filepath.Walk("web", func(fpath string, info os.FileInfo, err error) error {
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
		// no raw asset
		if strings.HasPrefix(fpath, "web/raw/") {
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
		fpath2 := "**" + fpath
		fpath2 = strings.ReplaceAll(fpath2, "**web/", "/")
		binMap[fpath2] = i

		// convert binary to base64
		b, err := ioutil.ReadFile(fpath)
		check(err)
		bs := base64.URLEncoding.EncodeToString(b)

		bf := BinFile{
			Name:    info.Name(),
			Size:    info.Size(),
			Mode:    info.Mode(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
			Data64:  bs,
		}

		fptr.WriteString(fmt.Sprintf("var __binfile%d = ", i))

		td, terr := template.New(fmt.Sprintf("data%d", i)).Parse(binfileTemplate)
		tmpl := template.Must(td, terr)
		err = tmpl.Execute(fptr, bf)
		check(err)

		bdat := bf.Data64
		max := int(math.Ceil(float64(len(bdat)) / 100))
		fptr.WriteString(`Data64: `)
		for i := 0; i < max; i++ {
			head := i * 100
			tail := (i + 1) * 100
			if tail > len(bdat) {
				tail = len(bdat)
			}

			if i == 0 && max == 1 {
				fptr.WriteString(`"` + bdat[head:tail] + `",`)
			} else if i == 0 {
				fptr.WriteString(`"` + bdat[head:tail] + `" +`)
			} else if i == max-1 {
				fptr.WriteString("\t\t" + `"` + bdat[head:tail] + `",`)
			} else {
				fptr.WriteString("\t\t" + `"` + bdat[head:tail] + `" +`)
			}
			fptr.WriteString("\n")
		}
		fptr.WriteString("}\n\n")

		fmt.Println(i, bf.Size, fpath)

		return nil
	})
	check(err)

	td, terr := template.New(fmt.Sprintf("map%d", i)).Parse(binmapTemplate)
	tmpl := template.Must(td, terr)
	err = tmpl.Execute(fptr, binMap)
	check(err)

	fptr.WriteString("}")
	fptr.Close()

	fmt.Println("done")
}
