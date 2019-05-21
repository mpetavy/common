package common

import (
	"flag"
	"github.com/disintegration/imaging"
	"image"
	"image/draw"
	"io"
	"os"
)

var quality *int

type Orientation int

const (
	ORIENTATION_0 Orientation = iota
	ORIENTATION_90
	ORIENTATION_180
	ORIENTATION_270
)

type Rotation int

const (
	ROTATE_0 Rotation = iota
	ROTATE_90
	ROTATE_180
	ROTATE_270
)

func init() {
	quality = flag.Int("jpeg.quality", 80, "JPEG quality")
}

func LoadImage(path string) (image.Image, error) {
	return imaging.Open(path)
}

func CopyImage(src image.Image) draw.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)

	return dst
}

func Rotate(source image.Image, rotation Rotation) image.Image {
	switch rotation {
	case ROTATE_0:
		return CopyImage(source)
	case ROTATE_90:
		return imaging.Rotate90(source)
	case ROTATE_180:
		return imaging.Rotate180(source)
	case ROTATE_270:
		return imaging.Rotate270(source)
	}

	return source
}

func EncodeJpeg(source image.Image, w io.Writer) error {
	return imaging.Encode(w, source, imaging.JPEG, imaging.JPEGQuality(*quality))
}

func SaveJpeg(source image.Image, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	return EncodeJpeg(source, f)
}
