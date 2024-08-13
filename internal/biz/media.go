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
	"github.com/nats-io/nats.go"
	"google.golang.org/genproto/googleapis/api/httpbody"

	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/data"
	u_nats "gitlab.calendaria.team/services/utils/v1/nats"
	u_jwt "gitlab.calendaria.team/services/utils/v2/jwt"
)

// MediaUsecase is a Greeter usecase.
type MediaUsecase struct {
	log       *log.Helper
	jwt       u_jwt.IJwtProcessor
	mediaRepo data.MediaRepo
	s3        *data.S3Uploader
	qm        u_nats.IQueueManager
}

// NewGreeterUsecase new a Greeter usecase.
func NewMediaUsecase(
	logger log.Logger,
	jwt u_jwt.IJwtProcessor,
	mediaRepo data.MediaRepo,
	s3 *data.S3Uploader,
	qm u_nats.IQueueManager,
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

func (uc *MediaUsecase) deleteMediaConsumer(ctx context.Context, m *nats.Msg) bool {
	var mediaID int64
	err := json.Unmarshal(m.Data, &mediaID)
	if err != nil {
		uc.log.Errorf("deleteMediaConsumer: json.Unmarshal: %s", err.Error())
		return true
	}

	media, err := uc.mediaRepo.GetMedia(ctx, mediaID)
	if err != nil {
		return false
	}

	if uc.s3.Session == nil {
		return true
	}

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

	return true
}

func getExtension(contentType string) (string, bool) {
	extension, ok := allowedContentTypesConst[contentType]
	if ok {
		return extension, true
	}

	return "", false
}

func (uc *MediaUsecase) UploadMedia(
	ctx context.Context, userID int64, fileName, filePath string, file *httpbody.HttpBody, isPrivate bool,
) (*ent.Media, error) {
	contentType := file.GetContentType()
	extension, ok := getExtension(contentType)
	if !ok {
		return nil, v1.ErrorInvalidContentType("Invalid content type: %s", contentType)
	}

	path := filePath
	if path == "" {
		uid := uuid.NewString()
		path = fmt.Sprintf("%d/%s/%s.%s", userID, time.Now().Format("2006/01"), uid, extension)
	} else {
		path += fileName
	}

	createMediaDto := data.CreateMediaDto{
		OwnerID:   userID,
		FileName:  fileName,
		Path:      path,
		Extension: extension,
		Size:      int32(len(file.GetData())),
		IsPrivate: isPrivate,
	}
	media, err := uc.mediaRepo.CreateMedia(ctx, createMediaDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("CreateMedia error: %s", err.Error())
	}

	location, err := uc.s3.Upload(ctx, media.Path, file.GetData(), file.GetContentType(), isPrivate)
	if err != nil {
		_ = uc.mediaRepo.DeleteMedia(ctx, media.ID)

		return nil, v1.ErrorS3uploadFailed("S3 Upload error: %s", err.Error())
	}

	media, err = uc.mediaRepo.SetMediaLocation(ctx, media, location)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("SetMediaUploadedAt error: %s", err.Error())
	}

	go func() {
		_ = uc.appendMedia(ctx, userID, media, file, isPrivate)
	}()

	return media, nil
}

func (uc *MediaUsecase) appendMedia(
	_ context.Context,
	userID int64,
	media *ent.Media,
	file *httpbody.HttpBody,
	isPrivate bool,
) error {
	var err error

	contentType := file.GetContentType()
	re := regexp.MustCompile(`^(.*)\/.*`)

	format := re.FindStringSubmatch(contentType)
	if len(format) == 0 {
		uc.log.Error(v1.ErrorInvalidContentType("invalid content type: %s", contentType))

		return err
	}

	switch format[1] {
	case "video":
		err = uc.appendVideo(file, userID, media, isPrivate)
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

func (uc *MediaUsecase) appendVideo(file *httpbody.HttpBody, userID int64, media *ent.Media, isPrivate bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), data.DefaultTimeout)
	defer cancel()

	meta, thumbnail, err := uc.extractVideoInfo(ctx, file)
	if err != nil {
		return err
	}

	err = uc.setVideoParams(ctx, userID, media, thumbnail, meta, isPrivate)
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
	ctx, cancel := context.WithTimeout(context.Background(), data.DefaultTimeout)
	defer cancel()

	contentType := file.GetContentType()
	extension, ok := getExtension(contentType)
	if !ok {
		err := v1.ErrorInvalidContentType("incorect type: %s", file.GetContentType())

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
		err = v1.ErrorInternal("vp.ProcessVideo: write video error: %s", err.Error())

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
			errs = errors.Join(errs, fmt.Errorf("tg.GetMetadata error: %s", err.Error()))
			mx.Unlock()
		}
	}()

	go func() {
		thumbnail, err = tg.GetThumbnail()
		if err != nil {
			mx.Lock()
			errs = errors.Join(errs, fmt.Errorf("tg.GetThumbnail error: %s", err.Error()))
			mx.Unlock()
		}
	}()

	err = tg.Wait()
	if err != nil {
		mx.Lock()
		errs = errors.Join(errs, fmt.Errorf("tg.Wait error: %s", err.Error()))
		mx.Unlock()
	}

	if errs != nil {
		return "", nil, v1.ErrorInternal("uc.extractVideoInfo error: %s", errs.Error())
	}

	return meta, thumbnail, nil
}

func (uc *MediaUsecase) setVideoParams(
	ctx context.Context,
	userID int64,
	media *ent.Media,
	thumbnail *data.Image,
	meta string,
	isPrivate bool,
) error {
	uuid := uuid.NewString()
	path := fmt.Sprintf("%d/%s/%s.%s", userID, time.Now().Format("2006/01"), uuid, thumbnail.Extension)

	location, err := uc.s3.Upload(ctx, path, thumbnail.Data, thumbnail.MimeType, isPrivate)
	if err != nil {
		_ = uc.s3.Delete(ctx, path)
		err = v1.ErrorS3uploadFailed("S3 Upload error: %s", err)

		return err
	}

	duration, err := data.ExtractDurationFromMetadata(meta)
	if err != nil {
		_ = uc.s3.Delete(ctx, path)
		err := v1.ErrorInternal("uc.uploadThumbnail: ExtractDurationFromMetadata error: %s", err.Error())

		return err
	}

	_, err = uc.mediaRepo.SetVideoParameters(
		ctx,
		media,
		data.SetVideoParamsDto{
			Duration:      duration,
			ThumbnailURL:  location,
			ThumbnailPath: path,
		},
	)
	if err != nil {
		_ = uc.s3.Delete(ctx, path)
		err = v1.ErrorDatabaseQuery("uc.uploadThumbnail: SetVideoParameters error: %s", err.Error())

		return err
	}

	return nil
}

func (uc *MediaUsecase) setMediaDimensions(ctx context.Context, media *ent.Media, img *data.Image) error {
	width, height, err := img.GetDimensions()
	if err != nil {
		err = v1.ErrorInternal("uc.setMediaDimensions: GetDimensions error: %s", err.Error())

		return err
	}

	setDimensionsDto := data.SetDimensionsDto{Width: width, Height: height}

	_, err = uc.mediaRepo.SetDimensions(ctx, media, setDimensionsDto)
	if err != nil {
		err = v1.ErrorDatabaseQuery("uc.setMediaDimensions: SetDimensions error: %s", err.Error())

		return err
	}

	return nil
}

func (uc *MediaUsecase) GetMedia(ctx context.Context, mediaID int64) (*ent.Media, error) {
	media, err := uc.mediaRepo.GetMedia(ctx, mediaID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("Media not found")
		}
		return nil, v1.ErrorDatabaseQuery("GetMedia error: %s", err.Error())
	}

	var url string

	if media.IsPrivate && media.URL != nil {
		url, err = uc.s3.GetPresignedURL(ctx, media.Path)
		if err != nil {
			return nil, v1.ErrorS3Failed("S3 GetPresignedURL error: %s", err.Error())
		}

		media.URL = &url
	}

	if media.IsPrivate && media.ThumbnailURL != nil {
		url, err = uc.s3.GetPresignedURL(ctx, *media.ThumbnailPath)
		if err != nil {
			return nil, v1.ErrorS3Failed("S3 GetPresignedURL error: %s", err.Error())
		}

		media.ThumbnailPath = &url
	}

	return media, nil
}

func (uc *MediaUsecase) GetMediaList(ctx context.Context, userID int64, ownOnly bool, mediaIDs []int64) (
	[]*ent.Media, error,
) {
	filter := data.FilterMediaDto{
		MediaIDs: mediaIDs,
	}

	if ownOnly {
		filter.OwnerID = &userID
	}

	mediaList, err := uc.mediaRepo.GetMediaList(ctx, filter)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("GetMediaList error: %s", err.Error())
	}

	return mediaList, nil
}
