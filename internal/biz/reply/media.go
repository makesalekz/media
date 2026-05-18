package reply

import (
	"time"

	v1 "github.com/makesalekz/media/api/media/v1"
	"github.com/makesalekz/media/ent"
)

func MapMedia(media *ent.Media) *v1.Media {
	result := &v1.Media{
		Id:        media.ID,
		OwnerId:   media.OwnerID,
		FileName:  media.FileName,
		Extension: media.Extension,
		Size:      media.Size,
		Width:     media.Width,
		Height:    media.Height,
		Duration:  media.Duration,
		CreatedAt: media.CreatedAt.Format(time.RFC3339),
	}
	if media.URL != nil {
		result.Url = *media.URL
	}

	if media.ThumbnailURL != nil {
		result.ThumbnailUrl = media.ThumbnailURL
	}

	return result
}
