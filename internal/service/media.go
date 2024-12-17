package service

import (
	"context"
	"net/url"
	"time"

	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/biz"
	"gitlab.calendaria.team/services/media/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
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
	actorID := auth.GetActorIdFromContext(ctx)
	if actorID == 0 {
		return nil, v1.ErrorEmptyActorId("empty actor id")
	}
	appID := auth.GetAppIdFromContext(ctx)
	if appID == "" {
		return nil, v1.ErrorEmptyActorId("empty app id")
	}

	fileName, err := url.QueryUnescape(req.GetFileName())
	if err != nil {
		return nil, v1.ErrorInvalidRequest("invalid file name: %s", req.GetFileName())
	}

	media, err := s.uc.UploadMedia(ctx, actorID, fileName, req.GetFilePath(), req.GetContent(), req.GetIsPrivate())
	if err != nil {
		return nil, err
	}

	return &v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *MediaService) GetMedia(ctx context.Context, req *v1.GetMediaRequest) (*v1.MediaReply, error) {
	media, err := s.uc.GetMedia(ctx, req.GetMediaId())
	if err != nil {
		return nil, err
	}

	return &v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *MediaService) GetMediaList(ctx context.Context, req *v1.GetMediaListRequest) (*v1.MediaListReply, error) {
	filter := data.FilterMediaDto{
		MediaIDs: req.GetMediaIds(),
	}

	if req.GetOwnOnly() {
		actorID := auth.GetActorIdFromContext(ctx)
		if actorID == 0 {
			return nil, v1.ErrorEmptyActorId("empty actor id")
		}

		filter.OwnerID = &actorID
	}

	mediaList, err := s.uc.GetMediaList(ctx, filter)
	if err != nil {
		return nil, err
	}

	resultList := make([]*v1.Media, len(mediaList))
	for i, media := range mediaList {
		resultList[i] = replyMedia(media)
	}

	return &v1.MediaListReply{Media: resultList}, nil
}

func (s *MediaService) DeleteAvatar(ctx context.Context, req *v1.DeleteAvatarRequest) (*utils_v1.EmptyReply, error) {
	err := s.uc.DeleteAvatar(ctx, req.GetUrls())
	if err != nil {
		return nil, err
	}

	return &utils_v1.EmptyReply{}, nil
}
