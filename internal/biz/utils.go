package biz

import (
	"regexp"

	data_types "gitlab.calendaria.team/services/media/internal/data"
)

// GetExtension возвращает расширение файла по типу контента
func GetExtension(contentType string) (string, bool) {
	extension, ok := allowedContentTypesConst[contentType]
	if ok {
		return extension, true
	}

	return "", false
}

// ParseContentType извлекает базовый тип контента (видео, изображение, аудио и т.д.)
func ParseContentType(contentType string) (string, bool) {
	re := regexp.MustCompile(`^(.*)\/.*`)
	format := re.FindStringSubmatch(contentType)

	if len(format) == 0 {
		return "", false
	}

	return format[1], true
}

// IsImageType проверяет, является ли тип контента изображением
func IsImageType(contentType string) bool {
	baseType, ok := ParseContentType(contentType)
	if !ok {
		return false
	}
	return baseType == "image"
}

// IsVideoType проверяет, является ли тип контента видео
func IsVideoType(contentType string) bool {
	baseType, ok := ParseContentType(contentType)
	if !ok {
		return false
	}
	return baseType == "video"
}

// IsAudioType проверяет, является ли тип контента аудио
func IsAudioType(contentType string) bool {
	baseType, ok := ParseContentType(contentType)
	if !ok {
		return false
	}
	return baseType == "audio"
}

// CreateImage создает структуру Image из необработанных данных
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
