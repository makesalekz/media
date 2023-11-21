package biz

import "github.com/google/wire"

const (
	QueueDeleteMedia = "delete"
)

var (
	allowedContentTypesConst = map[string]string{
		"image/jpeg":         "jpg",
		"image/png":          "png",
		"image/gif":          "gif",
		"image/webp":         "webp",
		"image/bmp":          "bmp",
		"image/vnd.wap.wbmp": "wbmp",

		"video/x-msvideo": "avi",
		"video/mp4":       "mp4",
		"video/webm":      "webm",
		"video/3gpp":      "3gp",
		"video/3gpp2":     "3g2",
		"video/qucktime":  "mov",
	}
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewQueueManager, NewMediaUsecase)
