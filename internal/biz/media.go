package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	nnats "github.com/nats-io/nats.go"
	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/data"
	jwtp "gitlab.calendaria.team/services/utils/v1/jwt"
	"gitlab.calendaria.team/services/utils/v1/nats"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// MediaUsecase is a Greeter usecase.
type MediaUsecase struct {
	log       *log.Helper
	jwt       *jwtp.JwtProcessor
	mediaRepo data.MediaRepo
	s3        *data.S3Uploader
	qm        *nats.QueueManager
}

// NewGreeterUsecase new a Greeter usecase.
func NewMediaUsecase(
	logger log.Logger,
	jwt *jwtp.JwtProcessor,
	mediaRepo data.MediaRepo,
	s3 *data.S3Uploader,
	qm *nats.QueueManager,
) (*MediaUsecase, error) {
	uc := &MediaUsecase{
		log:       log.NewHelper(logger),
		jwt:       jwt,
		mediaRepo: mediaRepo,
		s3:        s3,
		qm:        qm,
	}

	qm.AddConsumer(QueueDeleteMedia, uc.deleteMediaConsumer)

	return uc, nil
}

func (uc *MediaUsecase) deleteMediaConsumer(ctx context.Context, m *nnats.Msg) bool {
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

	if uc.s3.Session != nil {
		err = uc.s3.Delete(ctx, media.Path)
		if err != nil {
			return false
		}

		if media.ThumbnailPath != nil {
			if *media.ThumbnailPath != "" {
				err = uc.s3.Delete(ctx, *media.ThumbnailPath)
				if err != nil {
					return false
				}
			}
		}

		err = uc.mediaRepo.DeleteMedia(ctx, media.ID)
		if err != nil {
			return true
		}
	}

	return true
}

func getExtension(contentType string) (string, bool) {
	extension, ok := allowedContentTypesConst[contentType]
	if ok {
		return extension, true
	}

	return "", false
}

func (uc *MediaUsecase) UploadMedia(ctx context.Context, userId int64, fileName, filePath string, file *httpbody.HttpBody) (*ent.Media, error) {
	contentType := file.GetContentType()
	extension, ok := getExtension(contentType)
	if !ok {
		return nil, v1.ErrorInvalidContentType("Invalid content type: %s", contentType)
	}

	path := filePath
	if path == "" {
		uuid := uuid.NewString()
		path = fmt.Sprintf("%d/%s/%s.%s", userId, time.Now().Format("2006/01"), uuid, extension)
	} else {
		path += fileName
	}

	createMediaDto := data.CreateMediaDto{
		OwnerId:   userId,
		FileName:  fileName,
		Path:      path,
		Extension: extension,
		Size:      int32(len(file.GetData())),
	}
	media, err := uc.mediaRepo.CreateMedia(ctx, createMediaDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("CreateMedia error: %s", err)
	}

	location, err := uc.s3.Upload(ctx, media.Path, file.GetData(), file.GetContentType())
	if err != nil {
		_ = uc.mediaRepo.DeleteMedia(ctx, media.ID)

		return nil, v1.ErrorS3uploadFailed("S3 Upload error: %s", err)
	}

	media, err = uc.mediaRepo.SetMediaLocation(ctx, media, location)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("SetMediaUploadedAt error: %s", err)
	}

	go func() {
		_ = uc.appendMedia(ctx, userId, media, file)
	}()

	return media, nil
}

func (uc *MediaUsecase) appendMedia(
	_ context.Context,
	userId int64,
	media *ent.Media,
	file *httpbody.HttpBody,
) error {
	contentType := file.GetContentType()
	re, err := regexp.Compile(`^(.*)\/.*`)
	if err != nil {
		uc.log.Error(err)

		return err
	}

	format := re.FindStringSubmatch(contentType)
	if len(format) == 0 {
		uc.log.Error(v1.ErrorInvalidContentType("invalid content type: %s", contentType))

		return err
	}

	switch format[1] {
	case "video":
		err = uc.appendVideo(file, userId, media)
		if err != nil {
			uc.log.Error(err)

			return err
		}
	case "image":
		err = uc.appendImage(file, media)
		if err != nil {
			uc.log.Error(err)

			return err
		}
	}

	return nil
}

func (uc *MediaUsecase) appendVideo(file *httpbody.HttpBody, userId int64, media *ent.Media) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	meta, thumbnail, err := uc.extractVideoInfo(ctx, file)
	if err != nil {
		return err
	}

	err = uc.setVideoParams(ctx, userId, media, thumbnail, meta)
	if err != nil {
		return err
	}

	err = uc.setMediaDimensions(ctx, media, thumbnail)
	if err != nil {
		return err
	}

	return nil
}

func (uc *MediaUsecase) appendImage(file *httpbody.HttpBody, media *ent.Media) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	contentType := file.GetContentType()
	extension, ok := getExtension(contentType)
	if !ok {
		err := v1.ErrorInvalidContentType("incorect type: %v", file.ContentType)

		return err
	}

	img := &data.Image{
		Data:      file.GetData(),
		Extension: extension,
		MimeType:  contentType,
	}

	err := uc.setMediaDimensions(ctx, media, img)
	if err != nil {
		return err
	}

	return nil
}

func (uc *MediaUsecase) extractVideoInfo(ctx context.Context, file *httpbody.HttpBody) (string, *data.Image, error) {
	vp, err := data.NewVideoProcessor(ctx)
	if err != nil {
		return "", nil, err
	}

	tg, err := vp.GetThumbnailGenerator(file.GetData())
	if err != nil {
		err = v1.ErrorInternal("vp.ProcessVideo: write video error: %v", err)

		return "", nil, err
	}
	defer tg.Close()

	err = tg.Start()
	if err != nil {
		return "", nil, err
	}

	var meta string
	var thumbnail *data.Image
	var errs error
	mx := sync.Mutex{}

	go func() {
		meta, err = tg.GetMetadata()
		if err != nil {
			mx.Lock()
			errs = errors.Join(errs, fmt.Errorf("tg.GetMetadata error: %v", err))
			mx.Unlock()
		}
	}()

	go func() {
		thumbnail, err = tg.GetThumbnail()
		if err != nil {
			mx.Lock()
			errs = errors.Join(errs, fmt.Errorf("tg.GetThumbnail error: %v", err))
			mx.Unlock()
		}
	}()

	err = tg.Wait()
	if err != nil {
		mx.Lock()
		errs = errors.Join(errs, fmt.Errorf("tg.Wait error: %v", err))
		mx.Unlock()
	}

	if errs != nil {
		return "", nil, v1.ErrorInternal("uc.extractVideoInfo error: %v", errs)
	}

	return meta, thumbnail, nil
}

func (uc *MediaUsecase) setVideoParams(
	ctx context.Context,
	userId int64,
	media *ent.Media,
	thumbnail *data.Image,
	meta string,
) error {
	uuid := uuid.NewString()
	path := fmt.Sprintf("%d/%s/%s.%s", userId, time.Now().Format("2006/01"), uuid, thumbnail.Extension)

	location, err := uc.s3.Upload(ctx, path, thumbnail.Data, thumbnail.MimeType)
	if err != nil {
		_ = uc.s3.Delete(ctx, path)
		err = v1.ErrorS3uploadFailed("S3 Upload error: %s", err)

		return err
	}

	duration, err := data.ExtractDurationFromMetadata(meta)
	if err != nil {
		_ = uc.s3.Delete(ctx, path)
		err := v1.ErrorInternal("uc.uploadThumbnail: ExtractDurationFromMetadata error: %v", err)

		return err
	}

	_, err = uc.mediaRepo.SetVideoParameters(
		ctx,
		media,
		data.SetVideoParamsDto{
			Duration:      duration,
			ThumbnailUrl:  location,
			ThumbnailPath: path,
		},
	)
	if err != nil {
		_ = uc.s3.Delete(ctx, path)
		err = v1.ErrorDatabaseQuery("uc.uploadThumbnail: SetVideoParameters error: %s", err)

		return err
	}

	return nil
}

func (uc *MediaUsecase) setMediaDimensions(ctx context.Context, media *ent.Media, img *data.Image) error {
	width, height, err := img.GetDimensions()
	if err != nil {
		err = v1.ErrorInternal("uc.setMediaDimensions: GetDimensions error: %s", err)

		return err
	}

	setDimensionsDto := data.SetDimensionsDto{Width: width, Height: height}

	_, err = uc.mediaRepo.SetDimensions(ctx, media, setDimensionsDto)
	if err != nil {
		err = v1.ErrorDatabaseQuery("uc.setMediaDimensions: SetDimensions error: %s", err)

		return err
	}

	return nil
}

func (uc *MediaUsecase) GetMedia(ctx context.Context, mediaId int64) (*ent.Media, error) {
	media, err := uc.mediaRepo.GetMedia(ctx, mediaId)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("Media not found")
		}
		return nil, v1.ErrorDatabaseQuery("GetMedia error: %s", err)
	}

	return media, nil
}

func (uc *MediaUsecase) GetMediaList(ctx context.Context, userId int64, ownOnly bool, mediaIds []int64) ([]*ent.Media, error) {
	filter := data.FilterMediaDto{
		MediaIds: mediaIds,
	}

	if ownOnly {
		filter.OwnerId = &userId
	}

	mediaList, err := uc.mediaRepo.GetMediaList(ctx, filter)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("GetMediaList error: %s", err)
	}

	return mediaList, nil
}
