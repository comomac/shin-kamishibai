package main

// this program creates binfile.go
// which contains assets to help compile single executable

import (
	"bytes"
	"compress/lzw"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const binfileTemplate = `&BinFile{
	Name:         "{{.Name}}",
	Size:         {{.Size}},
	Mode:         0644,
	ModTime:      time.Unix({{.ModTime.Unix}}, 0),
	IsDir:        {{.IsDir}},
	IsCompressed: {{.IsCompressed}},
	Compression:  "{{.Compression}}",
	`

const binmapTemplate = `var __binmapName = map[string]*BinFile{
{{range $k, $v := .}}{{printf "\t"}}"{{$k}}": __binfile{{$v}},
{{end}}`

// BinFile copys struct of os.FileInfo to mimic fake file system
type BinFile struct {
	Name         string
	Size         int64
	Mode         os.FileMode
	ModTime      time.Time
	IsDir        bool
	Sys          interface{}
	ready        bool   // is the data filled and ready to serve?
	data         []byte // used during program runtime
	IsCompressed bool   // is it compressed?
	Compression  string // what kind of compression algorithm and setting
	Data64       string // used during generate
}

var allowedExt = []string{"jpg", "jpeg", "png", "gif", "htm", "html", "css", "js"}
var compressExt = []string{"htm", "html", "css", "js", "json", "txt", "md"}

func main() {
	fmt.Println("building binfile.go ...")

	var err error

	tmplStr, err := ioutil.ReadFile("cmd/gen/binfile.go.tmpl")
	if err != nil {
		panic(err)
	}

	fptr, err := os.OpenFile("binfile.go", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	_, err = fptr.Write(tmplStr)
	if err != nil {
		panic(err)
	}

	// start blank
	binMap := map[string]int{}

	// file counter
	i := 0

	// add folders
	wdirs := []string{"web", "ssp"}
	for _, wdir := range wdirs {
		err = filepath.Walk(wdir, func(fpath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// revert windows path to / instead of \
			fpath = strings.Replace(fpath, `\`, "/", -1)
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
			if wdir == "web" {
				// web needs to be at the root level to work with http.FileServe
				fpath2 := "**" + fpath
				fpath2 = strings.Replace(fpath2, "**web/", "/", -1)
				binMap[fpath2] = i
			} else {
				binMap[fpath] = i
			}

			err = addFile(i, fpath, info, fptr)
			if err != nil {
				panic(err)
			}

			fmt.Println(i, info.Size(), fpath)

			return nil
		})
		if err != nil {
			panic(err)
		}
	}

	// add files
	wfiles := []string{
		"sample-conf.json",
	}
	for _, wfile := range wfiles {
		ff, err := os.Open(wfile)
		if err != nil {
			panic(err)
		}
		fstat, err := ff.Stat()
		if err != nil {
			panic(err)
		}

		i++
		binMap[wfile] = i

		err = addFile(i, wfile, fstat, fptr)
		if err != nil {
			panic(err)
		}

		fmt.Println(i, fstat.Size(), wfile)
	}

	td, terr := template.New(fmt.Sprintf("map%d", i)).Parse(binmapTemplate)
	tmpl := template.Must(td, terr)
	err = tmpl.Execute(fptr, binMap)
	if err != nil {
		panic(err)
	}

	_, err = fptr.WriteString("}")
	if err != nil {
		panic(err)
	}
	err = fptr.Close()
	if err != nil {
		panic(err)
	}

	fmt.Println("done")
}

func addFile(fNum int, fpath string, info os.FileInfo, fptr *os.File) error {
	// check to compress
	compress := false
	compression := ""
	for _, ext := range compressExt {
		if strings.HasSuffix(strings.ToLower(info.Name()), "."+ext) {
			compress = true
			break
		}
	}

	// read file
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if compress {
		// compress
		compression = "lzw-msb-8"
		w := lzw.NewWriter(&buf, lzw.MSB, 8)
		_, err = w.Write(b)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
	} else {
		buf = *bytes.NewBuffer(b)
	}

	// convert data to base64
	b64 := base64.URLEncoding.EncodeToString(buf.Bytes())

	bf := BinFile{
		Name:         info.Name(),
		Size:         info.Size(),
		Mode:         info.Mode(),
		ModTime:      info.ModTime(),
		IsDir:        info.IsDir(),
		Data64:       b64,
		IsCompressed: compress,
		Compression:  compression,
	}

	fptr.WriteString(fmt.Sprintf("var __binfile%d = ", fNum))

	td, terr := template.New(fmt.Sprintf("data%d", fNum)).Parse(binfileTemplate)
	tmpl := template.Must(td, terr)
	err = tmpl.Execute(fptr, bf)
	if err != nil {
		return err
	}

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

	return nil
}
