package service

import (
	"context"
	"media/ent"

	media_v1 "media/api/media/v1"
	"media/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type MediaService struct {
	media_v1.UnimplementedMediaServiceServer

	log *log.Helper
	uc  *biz.MediaUsecase
}

func NewMediaService(logger log.Logger, uc *biz.MediaUsecase) *MediaService {
	return &MediaService{
		log: log.NewHelper(logger),
		uc:  uc,
	}
}

func replyMedia(media *ent.Media) *media_v1.Media {
	result := &media_v1.Media{
		Id:        media.ID,
		OwnerId:   media.OwnerID,
		FileName:  media.FileName,
		Extension: media.Extension,
		Size:      media.Size,
		Width:     media.Width,
		Height:    media.Height,
		Duration:  media.Duration,
	}
	if media.URL != nil {
		result.Url = *media.URL
	}
	return result
}

func (s *MediaService) UploadMedia(ctx context.Context, req *media_v1.UploadMediaRequest) (*media_v1.MediaReply, error) {
	media, err := s.uc.UploadMedia(ctx, req.FileName, req.Content)
	if err != nil {
		return nil, err
	}

	return &media_v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *MediaService) GetMedia(ctx context.Context, req *media_v1.GetMediaRequest) (*media_v1.MediaReply, error) {
	media, err := s.uc.GetMedia(ctx, req.MediaId)
	if err != nil {
		return nil, err
	}

	return &media_v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *MediaService) GetMediaList(ctx context.Context, req *media_v1.GetMediaListRequest) (*media_v1.MediaListReply, error) {
	mediaList, err := s.uc.GetMediaList(ctx, req.OwnOnly, req.MediaIds)
	if err != nil {
		return nil, err
	}

	resultList := make([]*media_v1.Media, len(mediaList))
	for i, media := range mediaList {
		resultList[i] = replyMedia(media)
	}

	return &media_v1.MediaListReply{Media: resultList}, nil
}
