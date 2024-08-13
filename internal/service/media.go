package service

import (
	"context"
	"net/url"
	"time"

	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/biz"
	"gitlab.calendaria.team/services/utils/v2/auth"
)

type MediaService struct {
	v1.UnimplementedMediaServiceServer

	uc *biz.MediaUsecase
}

func NewMediaService(
	uc *biz.MediaUsecase,
) *MediaService {
	return &MediaService{
		uc: uc,
	}
}

func replyMedia(media *ent.Media) *v1.Media {
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

func (s *MediaService) UploadMedia(ctx context.Context, req *v1.UploadMediaRequest) (*v1.MediaReply, error) {
	actorId := auth.GetActorIdFromContext(ctx)
	if actorId == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}
	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyActorId("empty app id")
	}

	fileName, err := url.QueryUnescape(req.FileName)
	if err != nil {
		return nil, v1.ErrorInvalidRequest("invalid file name: %s", req.FileName)
	}

	media, err := s.uc.UploadMedia(ctx, actorId, fileName, req.FilePath, req.Content, req.GetIsPrivate())
	if err != nil {
		return nil, err
	}

	return &v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *MediaService) GetMedia(ctx context.Context, req *v1.GetMediaRequest) (*v1.MediaReply, error) {
	media, err := s.uc.GetMedia(ctx, req.MediaId)
	if err != nil {
		return nil, err
	}

	return &v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *MediaService) GetMediaList(ctx context.Context, req *v1.GetMediaListRequest) (*v1.MediaListReply, error) {
	actorId := auth.GetActorIdFromContext(ctx)
	if actorId == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}

	mediaList, err := s.uc.GetMediaList(ctx, actorId, req.OwnOnly, req.MediaIds)
	if err != nil {
		return nil, err
	}

	resultList := make([]*v1.Media, len(mediaList))
	for i, media := range mediaList {
		resultList[i] = replyMedia(media)
	}

	return &v1.MediaListReply{Media: resultList}, nil
}
