package biz

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"

	media_v1 "media/api/media/v1"
	"media/ent"
	"media/internal/conf"
	"media/internal/data"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// MediaUsecase is a Greeter usecase.
type MediaUsecase struct {
	conf      *conf.Bootstrap
	log       *log.Helper
	discovery *consul.Registry
	jwt       *data.JwtProcessor
	mediaRepo data.MediaRepo
	s3        *data.S3Uploader
	qm        *QueueManager
}

// NewGreeterUsecase new a Greeter usecase.
func NewMediaUsecase(
	logger log.Logger,
	c *data.Config,
	jwt *data.JwtProcessor,
	mediaRepo data.MediaRepo,
	s3 *data.S3Uploader,
	qm *QueueManager,
) (*MediaUsecase, error) {
	uc := &MediaUsecase{
		conf:      c.Bootstrap,
		log:       log.NewHelper(logger),
		discovery: c.GetRegistry(),
		jwt:       jwt,
		mediaRepo: mediaRepo,
		s3:        s3,
		qm:        qm,
	}

	qm.AddConsumer(QueueDeleteMedia, uc.deleteMediaConsumer)

	return uc, nil
}

func (uc *MediaUsecase) deleteMediaConsumer(ctx context.Context, m *nats.Msg) bool {
	var mediaId int64
	err := json.Unmarshal(m.Data, &mediaId)
	if err != nil {
		uc.log.Errorf("deleteMediaConsumer: json.Unmarshal: %s", err.Error())
		return true
	}

	media, err := uc.mediaRepo.GetMedia(ctx, mediaId)
	if err != nil {
		return false
	}

	err = uc.s3.Delete(ctx, media.Path)
	if err != nil {
		return false
	}

	err = uc.mediaRepo.DeleteMedia(ctx, media.ID)
	if err != nil {
		return true
	}

	return true
}

func getExtension(contentType string) (string, bool) {
	allowedContentTypes := []string{
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	for _, allowedContentType := range allowedContentTypes {
		if contentType == allowedContentType {
			return strings.Split(contentType, "/")[1], true
		}
	}

	return "", false
}

func (uc *MediaUsecase) UploadMedia(ctx context.Context, fileName string, file *httpbody.HttpBody) (*ent.Media, error) {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, media_v1.ErrorUnauthorized("Unauthorized")
	}

	contentType := file.GetContentType()
	extension, ok := getExtension(contentType)
	if !ok {
		return nil, media_v1.ErrorInvalidContentType("Invalid content type: %s", contentType)
	}

	media, err := uc.mediaRepo.CreateMedia(ctx, userId, fileName, extension, len(file.GetData()))
	if err != nil {
		return nil, media_v1.ErrorDatabaseQuery("CreateMedia error: %s", err)
	}

	location, err := uc.s3.Upload(ctx, media.Path, file)
	if err != nil {
		uc.mediaRepo.DeleteMedia(ctx, media.ID)

		return nil, media_v1.ErrorS3uploadFailed("S3 Upload error: %s", err)
	}

	media, err = uc.mediaRepo.SetMediaLocation(ctx, media, location)
	if err != nil {
		return nil, media_v1.ErrorDatabaseQuery("SetMediaUploadedAt error: %s", err)
	}

	return media, nil
}

func (uc *MediaUsecase) GetMedia(ctx context.Context, mediaId int64) (*ent.Media, error) {
	media, err := uc.mediaRepo.GetMedia(ctx, mediaId)
	if err != nil {
		return nil, media_v1.ErrorDatabaseQuery("GetMedia error: %s", err)
	}

	return media, nil
}

func (uc *MediaUsecase) GetMediaList(ctx context.Context, ownOnly bool, mediaIds []int64) ([]*ent.Media, error) {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, media_v1.ErrorUnauthorized("Unauthorized")
	}

	filter := data.FilterMediaDto{
		MediaIds: mediaIds,
	}

	if ownOnly {
		filter.OwnerId = &userId
	}

	mediaList, err := uc.mediaRepo.GetMediaList(ctx, filter)
	if err != nil {
		return nil, media_v1.ErrorDatabaseQuery("GetMediaList error: %s", err)
	}

	return mediaList, nil
}
