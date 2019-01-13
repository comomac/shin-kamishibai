package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi"
)

// KRoute encapsulate all the kamishibai route
type KRoute struct {
}

type responseErrorStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (k KRoute) responseError(w http.ResponseWriter, err error) {
	resp := &responseErrorStruct{
		Code:    http.StatusInternalServerError,
		Message: err.Error(),
	}

	str, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, string(str), http.StatusInternalServerError)
	}
}

func (k KRoute) bookInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "{}")
}

func (k KRoute) setBookmark(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "done")
}

func (k KRoute) renderThumbnail(w http.ResponseWriter, r *http.Request) {
	fp := chi.URLParam(r, "bookPath")

	zr, err := zip.OpenReader(fp)
	if err != nil {
		k.responseError(w, err)
		return
	}

	regexImageType := regexp.MustCompile(`(?i)\.(jpg|jpeg|gif|png)$`)
	var imgDat []byte
	for _, f := range zr.File {

		if regexImageType.MatchString(f.Name) {
			rc, err := f.Open()
			if err != nil {
				k.responseError(w, err)
				return
			}
			imgDat, err = imgThumb(rc)
			if err != nil {
				k.responseError(w, err)
				return
			}
			break
		}
	}

	if len(imgDat) > 0 {
		ctype := http.DetectContentType(imgDat)
		w.Header().Add("Content-Type", ctype)
		w.Header().Add("Content-Length", strconv.Itoa(len(imgDat)))
		w.Write(imgDat)
	} else {
		http.NotFound(w, r)
	}
}

func (k KRoute) getPage(w http.ResponseWriter, r *http.Request) {
	fp := chi.URLParam(r, "bookPath")
	page, _ := strconv.Atoi(chi.URLParam(r, "page"))

	// fp := "/Users/mac/tmp/RecoveryHD.sparsebundle.zip"

	fmt.Println("file", fp)

	zr, err := zip.OpenReader(fp)
	if err != nil {
		k.responseError(w, err)
		return
	}

	var ttlImages int
	regexImageType := regexp.MustCompile(`(?i)\.(jpg|jpeg|gif|png)$`)
	var imgDat []byte
	var imgFileName string
	for _, f := range zr.File {

		if regexImageType.MatchString(f.Name) {
			// fmt.Println("img!", f.Name)
			ttlImages++
			if page == ttlImages {
				imgFileName = f.Name

				rc, err := f.Open()
				if err != nil {
					k.responseError(w, err)
					return
				}
				imgDat, err = ioutil.ReadAll(rc)
				if err != nil {
					k.responseError(w, err)
					return
				}
			}
		} else {
			// fmt.Println("z..", f.Name, f.CompressedSize64, f.UncompressedSize64)
		}
	}

	// fmt.Println("found", ttlImages, "images")
	// fmt.Println(page, "th image name (", imgFileName, ")")

	// fmt.Fprint(w, "bytes")

	if imgFileName == "" {
		http.NotFound(w, r)
		return
	}

	// fmt.Printf("imgDat\n%+v\n", imgDat)
	ctype := http.DetectContentType(imgDat)
	w.Header().Add("Content-Type", ctype)
	w.Header().Add("Content-Length", strconv.Itoa(len(imgDat)))
	w.Write(imgDat)
}

func (k KRoute) listDir(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "{}")
}

func (k KRoute) deleteBook(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "deleted")
}
