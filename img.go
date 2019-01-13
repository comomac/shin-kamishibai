package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
)

func imgThumb(reader io.Reader) ([]byte, error) {
	return imgResize(reader, 320, 320)
}

func imgResize(reader io.Reader, owidth int, oheight int) ([]byte, error) {
	m, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	bounds := m.Bounds()

	// fmt.Printf("%+v", bounds)
	fmt.Println("dimension", bounds.Max.X, "x", bounds.Max.Y)

	// ratio
	var rx, ry float32
	rx = float32(owidth) / float32(bounds.Max.X)
	ry = float32(oheight) / float32(bounds.Max.Y)
	fmt.Println("ratio x, y", rx, ry)

	newImg := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{owidth, oheight},
		},
	)

	for x := 0; x < owidth; x++ {
		for y := 0; y < oheight; y++ {
			// imported image cord
			ix := int(float32(x) / rx)
			iy := int(float32(y) / ry)

			// r, g, b, a := m.At(x, y).RGBA()
			// rgba := m.At(x, y)
			rgba := m.At(ix, iy)

			// fmt.Println("xy:", x, ix, y, iy, rgba)

			newImg.Set(x, y, rgba)
			// newImg.Set(x, y, color.White)
		}
	}

	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	opts := &jpeg.Options{Quality: 50}
	jpeg.Encode(writer, newImg, opts)

	return b.Bytes(), nil
}
