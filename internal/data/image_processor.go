package data

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

type Image struct {
	Data      []byte
	Extension string
	MimeType  string
}

func (img *Image) GetDimensions() (int32, int32, error) {
	decodedImage, _, err := image.Decode(bytes.NewReader(img.Data))
	if err != nil {
		return 0, 0, err
	}

	return int32(decodedImage.Bounds().Dx()), int32(decodedImage.Bounds().Dy()), nil
}
