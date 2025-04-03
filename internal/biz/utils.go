package biz

import (
	"regexp"

	data_types "gitlab.calendaria.team/services/media/internal/data"
)

func GetExtension(contentType string) (string, bool) {
	extension, ok := allowedContentTypesConst[contentType]
	if ok {
		return extension, true
	}

	return "", false
}

func ParseContentType(contentType string) (string, bool) {
	re := regexp.MustCompile(`^(.*)\/.*`)
	format := re.FindStringSubmatch(contentType)

	if len(format) == 0 {
		return "", false
	}

	return format[1], true
}

func CreateImage(data []byte, contentType string) (*data_types.Image, bool) {
	extension, ok := GetExtension(contentType)
	if !ok {
		return nil, false
	}

	return &data_types.Image{
		Data:      data,
		Extension: extension,
		MimeType:  contentType,
	}, true
}
