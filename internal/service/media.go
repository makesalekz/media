package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/biz"
)

type MediaService struct {
	v1.UnimplementedMediaServiceServer

	log *log.Helper
	sh  *ServiceHelper
	uc  *biz.MediaUsecase
}

func NewMediaService(
	logger log.Logger,
	uc *biz.MediaUsecase,
	sh *ServiceHelper,
) *MediaService {
	return &MediaService{
		log: log.NewHelper(logger),
		sh:  sh,
		uc:  uc,
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
	return result
}

func (s *MediaService) UploadMedia(ctx context.Context, req *v1.UploadMediaRequest) (*v1.MediaReply, error) {
	actorId, err := s.sh.GetActorId(ctx, req.ActorId)
	if err != nil {
		return nil, err
	}

	media, err := s.uc.UploadMedia(ctx, actorId, req.FileName, req.FilePath, req.Content)
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
	actorId, err := s.sh.GetActorId(ctx, req.ActorId)
	if err != nil {
		return nil, err
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
