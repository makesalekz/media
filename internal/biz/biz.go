//nolint:gochecknoglobals // global variables used in wire and dict for content types
package biz

import (
	"github.com/google/wire"
	"github.com/makesalekz/utils/v2/nats"
)

const (
	QueueDeleteMedia       = "delete"
	QueueDeleteMediaList   = "delete_list"
	QueueDeleteMediaRecord = "delete_record"
)

var (
	allowedContentTypesConst = map[string]string{
		"image/jpeg":         "jpg",
		"image/png":          "png",
		"image/gif":          "gif",
		"image/webp":         "webp",
		"image/bmp":          "bmp",
		"image/vnd.wap.wbmp": "wbmp",

		"text/plain":    "txt",
		"text/yaml":     "yaml",
		"text/markdown": "md",
		"text/html":     "html",

		"application/rtf":               "rtf",
		"application/msword":            "doc",
		"application/pdf":               "pdf",
		"application/vnd.ms-powerpoint": "ppt",
		"application/json":              "json",
		"application/xml":               "xml",
		"application/xhtml+xml":         "xhtml",
		"application/csv":               "csv",
		"application/vnd.ms-excel":      "xls",

		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   "docx",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "xlsx",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "pptx",

		"audio/mpeg":  "mp3",
		"audio/wav":   "wav",
		"audio/ogg":   "ogg",
		"audio/x-m4a": "m4a",

		"video/mp4":        "mp4",
		"video/x-msvideo":  "avi",
		"video/quicktime":  "mov",
		"video/webm":       "webm",
		"video/3gpp":       "3gp",
		"video/3gpp2":      "3g2",
		"video/x-matroska": "mkv",

		"application/zip": "zip",
	}
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	nats.NewQueueManager,
	NewMediaUsecase,
)
