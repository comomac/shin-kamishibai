package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
)

func mkThumb(reader io.Reader) ([]byte, error) {
	m, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	bounds := m.Bounds()

	// fmt.Printf("%+v", bounds)
	fmt.Println("dimension", bounds.Max.X, "x", bounds.Max.Y)

	nx := 320
	ny := 320

	// ratio
	var rx, ry float32
	rx = float32(nx) / float32(bounds.Max.X)
	ry = float32(ny) / float32(bounds.Max.Y)
	fmt.Println("ratio x, y", rx, ry)

	newImg := image.NewRGBA(
		image.Rectangle{
			image.Point{0, 0},
			image.Point{nx, ny},
		},
	)

	for x := 0; x < nx; x++ {
		for y := 0; y < ny; y++ {
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
