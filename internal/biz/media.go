package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/data"
	"gitlab.calendaria.team/services/media/internal/data/dto"
	u_nats "gitlab.calendaria.team/services/utils/v2/nats"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// MediaUsecase .
type MediaUsecase struct {
	log       *log.Helper
	mediaRepo data.MediaRepo
	s3        data.S3Uploader
	qm        u_nats.IQueueManager
}

// NewMediaUsecase .
func NewMediaUsecase(
	logger log.Logger,
	mediaRepo data.MediaRepo,
	s3 data.S3Uploader,
	qm u_nats.IQueueManager,
) (*MediaUsecase, error) {
	uc := &MediaUsecase{
		log:       log.NewHelper(logger),
		mediaRepo: mediaRepo,
		s3:        s3,
		qm:        qm,
	}

	qm.AddConsumer(QueueDeleteMedia, uc.deleteMediaConsumer)
	qm.AddConsumer(QueueDeleteMediaList, uc.deleteMediaBulkConsumer)
	qm.AddConsumer(QueueDeleteMediaRecord, uc.deleteMediaRecordConsumer)

	return uc, nil
}

func (uc *MediaUsecase) deleteMediaConsumer(ctx context.Context, m jetstream.Msg) bool {
	var mediaID int64
	err := json.Unmarshal(m.Data(), &mediaID)
	if err != nil {
		uc.log.Errorf("deleteMediaConsumer: json.Unmarshal: %s", err.Error())
		return true
	}

	media, err := uc.mediaRepo.GetMedia(ctx, mediaID)
	if err != nil {
		uc.log.Errorf("deleteMediaConsumer: GetMedia: %s", err.Error())
		return false
	}

	if os.Getenv("DEBUG") == "" {
		err = uc.s3.Delete(ctx, media.Path)
		if err != nil {
			uc.log.Errorf("deleteMediaConsumer: Delete (path): %s", err.Error())
			return false
		}

		if media.ThumbnailPath != nil {
			if *media.ThumbnailPath != "" {
				err = uc.s3.Delete(ctx, *media.ThumbnailPath)
				if err != nil {
					uc.log.Errorf("deleteMediaConsumer: Delete (thumbnailPath): %s", err.Error())
					return false
				}
			}
		}
	}

	err = uc.mediaRepo.DeleteMedia(ctx, media.ID)
	if err != nil {
		// if there is an error on deleting media record in db, we need to requeue it
		uc.qm.GetLocal(QueueDeleteMediaRecord).Pub([]int64{mediaID})

		uc.log.Errorf("deleteMediaConsumer: mediaRepo.DeleteMedia: %s", err.Error())
		return true
	}

	return true
}

func (uc *MediaUsecase) deleteMediaBulkConsumer(ctx context.Context, m jetstream.Msg) bool {
	var mediaIDs []int64
	err := json.Unmarshal(m.Data(), &mediaIDs)
	if err != nil {
		uc.log.Errorf("deleteMediaConsumer: json.Unmarshal: %s", err.Error())
		return true
	}

	mediaList, err := uc.mediaRepo.ListMedia(ctx, mediaIDs)
	if err != nil {
		uc.log.Errorf("deleteMediaBulkConsumer: ListMedia: %s", err.Error())
		return false
	}

	if os.Getenv("DEBUG") == "" {
		paths := make([]string, len(mediaList))
		thumbnailPaths := make([]string, 0, len(mediaList))
		for i, media := range mediaList {
			paths[i] = media.Path

			if media.ThumbnailPath != nil {
				thumbnailPaths = append(thumbnailPaths, *media.ThumbnailPath)
			}
		}

		err = uc.s3.DeleteBulk(ctx, paths)
		if err != nil {
			uc.log.Errorf("deleteMediaBulkConsumer: DeleteBulk (paths): %s", err.Error())
			return false
		}

		if len(thumbnailPaths) > 0 {
			err = uc.s3.DeleteBulk(ctx, thumbnailPaths)
			if err != nil {
				uc.log.Errorf("deleteMediaBulkConsumer: DeleteBulk (thumbnailPaths): %s", err.Error())
				return false
			}
		}
	}

	_, err = uc.mediaRepo.DeleteMediaList(ctx, mediaIDs)
	if err != nil {
		// if there is an error on deleting media record in db, we need to requeue it
		uc.qm.GetLocal(QueueDeleteMediaRecord).Pub(mediaIDs)

		uc.log.Errorf("deleteMediaBulkConsumer: mediaRepo.DeleteMediaList: %s", err.Error())
		return true
	}

	return true
}

func (uc *MediaUsecase) deleteMediaRecordConsumer(ctx context.Context, m jetstream.Msg) bool {
	var mediaIDs []int64
	err := json.Unmarshal(m.Data(), &mediaIDs)
	if err != nil {
		uc.log.Errorf("deleteMediaConsumer: json.Unmarshal: %s", err.Error())
		return true
	}

	_, err = uc.mediaRepo.DeleteMediaList(ctx, mediaIDs)
	if err != nil {
		uc.log.Errorf("deleteMediaBulkConsumer: mediaRepo.DeleteMediaList: %s", err.Error())
		return false
	}

	return true
}

func (uc *MediaUsecase) UploadMedia(
	ctx context.Context, userID int64, fileName, filePath string, file *httpbody.HttpBody, isPrivate bool,
) (*ent.Media, error) {
	contentType := file.GetContentType()
	extension, ok := GetExtension(contentType)
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

	createMediaDto := dto.CreateMediaDto{
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

	mediaCopy := *media
	go func() {
		_ = uc.appendMedia(ctx, userID, &mediaCopy, file, isPrivate)
	}()

	media, err = uc.handlePrivateMedia(ctx, media)
	if err != nil {
		return nil, err
	}

	return media, nil
}

func (uc *MediaUsecase) handlePrivateMedia(ctx context.Context, media *ent.Media) (*ent.Media, error) {
	if !media.IsPrivate {
		return media, nil
	}

	var url string
	var err error

	if media.URL != nil {
		url, err = uc.s3.GetPreSignedURL(ctx, media.Path)
		if err != nil {
			return nil, v1.ErrorS3Failed("S3 GetPreSignedURL error: %s", err.Error())
		}

		media.URL = &url
	}

	if media.ThumbnailPath != nil {
		url, err = uc.s3.GetPreSignedURL(ctx, *media.ThumbnailPath)
		if err != nil {
			return nil, v1.ErrorS3Failed("S3 GetPreSignedURL error: %s", err.Error())
		}

		media.ThumbnailURL = &url
	}

	return media, nil
}

func (uc *MediaUsecase) GetMedia(ctx context.Context, mediaID int64) (*ent.Media, error) {
	media, err := uc.mediaRepo.GetMedia(ctx, mediaID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, v1.ErrorNotFound("Media not found")
		}
		return nil, v1.ErrorDatabaseQuery("GetMedia error: %s", err.Error())
	}

	media, err = uc.handlePrivateMedia(ctx, media)
	if err != nil {
		return nil, err
	}

	return media, nil
}

func (uc *MediaUsecase) GetMediaList(ctx context.Context, filter dto.FilterMediaDto) (
	[]*ent.Media, error,
) {
	mediaList, err := uc.mediaRepo.GetMediaList(ctx, filter)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("GetMediaList error: %s", err.Error())
	}

	return mediaList, nil
}

func (uc *MediaUsecase) DeleteAvatar(ctx context.Context, urls []string) error {
	mediaList, err := uc.mediaRepo.GetAvatarsMediaList(ctx, urls)
	if err != nil {
		return v1.ErrorDatabaseQuery("GetAvatarsMediaList error: %s", err)
	}

	mediaIDs := make([]int64, len(mediaList))
	for i, media := range mediaList {
		mediaIDs[i] = media.ID

		// delete avatar from aws s3
		err2 := uc.s3.Delete(ctx, media.Path)
		if err2 != nil {
			uc.log.Errorf("error on deleting avatar in aws: %s", err2)
		}
	}

	_, err = uc.mediaRepo.DeleteMediaList(ctx, mediaIDs)
	if err != nil {
		return v1.ErrorDatabaseQuery("DeleteMediaList error: %s", err)
	}

	return nil
}

func (uc *MediaUsecase) appendMedia(
	ctx context.Context,
	userID int64,
	media *ent.Media,
	file *httpbody.HttpBody,
	isPrivate bool,
) error {
	contentType := file.GetContentType()
	baseType, ok := ParseContentType(contentType)
	if !ok {
		uc.log.Error(v1.ErrorInvalidContentType("invalid content type: %s", contentType))
		return errors.New("invalid content type")
	}

	switch baseType {
	case "video":
		return uc.appendVideo(file, userID, media, isPrivate)
	case "image":
		return uc.appendImage(file, media)
	case "audio":
		return uc.appendAudio(file, media)
	default:
		return nil
	}
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
	img, ok := CreateImage(file.GetData(), contentType)
	if !ok {
		return v1.ErrorInvalidContentType("incorrect type: %s", file.GetContentType())
	}

	err := uc.setMediaDimensions(ctx, media, img)
	if err != nil {
		return err
	}

	return nil
}

func (uc *MediaUsecase) appendAudio(file *httpbody.HttpBody, media *ent.Media) error {
	ctx, cancel := context.WithTimeout(context.Background(), data.DefaultTimeout)
	defer cancel()

	duration, err := uc.extractAudioDuration(ctx, file)
	if err != nil {
		return err
	}

	_, err = uc.mediaRepo.SetDuration(ctx, media, duration)
	if err != nil {
		return v1.ErrorDatabaseQuery("uc.setAudioParams: SetAudioParameters error: %s", err.Error())
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
		err = v1.ErrorInternal("uc.uploadThumbnail: ExtractDurationFromMetadata error: %s", err.Error())

		return err
	}

	_, err = uc.mediaRepo.SetVideoParameters(
		ctx,
		media,
		dto.SetVideoParamsDto{
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

	setDimensionsDto := dto.SetDimensionsDto{Width: width, Height: height}

	_, err = uc.mediaRepo.SetDimensions(ctx, media, setDimensionsDto)
	if err != nil {
		err = v1.ErrorDatabaseQuery("uc.setMediaDimensions: SetDimensions error: %s", err.Error())

		return err
	}

	return nil
}

func (uc *MediaUsecase) extractAudioDuration(ctx context.Context, file *httpbody.HttpBody) (float32, error) {
	ap, err := data.NewAudioProcessor(ctx, file.GetData())
	if err != nil {
		return 0, err
	}
	defer ap.Close()

	err = ap.Start()
	if err != nil {
		return 0, err
	}

	var meta string
	var errs error
	mx := sync.Mutex{}

	go func() {
		meta, err = ap.GetMetadata()
		if err != nil {
			mx.Lock()
			errs = errors.Join(errs, fmt.Errorf("ap.GetMetadata error: %s", err.Error()))
			mx.Unlock()
		}
	}()

	err = ap.Wait()
	if err != nil {
		mx.Lock()
		errs = errors.Join(errs, fmt.Errorf("ap.Wait error: %s", err.Error()))
		mx.Unlock()
	}

	if errs != nil {
		return 0, v1.ErrorInternal("uc.extractAudioDuration error: %s", errs.Error())
	}

	duration, err := data.ExtractDurationFromMetadata(meta)
	if err != nil {
		return 0, err
	}

	return duration, nil
}
