package service

import (
	"context"
	"media/ent"

	upload_v1 "media/api/upload/v1"
	"media/internal/biz"
	"media/internal/data"

	"github.com/go-kratos/kratos/v2/log"
)

type UploadService struct {
	upload_v1.UnimplementedUploadServer

	log *log.Helper
	jwt *data.JwtProcessor
	uc  *biz.MediaUsecase
}

func NewUploadService(logger log.Logger, jwt *data.JwtProcessor, uc *biz.MediaUsecase) *UploadService {
	return &UploadService{
		log: log.NewHelper(logger),
		jwt: jwt,
		uc:  uc,
	}
}

func replyMedia(media *ent.Media) *upload_v1.Media {
	result := &upload_v1.Media{
		Id:        media.ID,
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

func (s *UploadService) UploadMedia(ctx context.Context, req *upload_v1.UploadMediaRequest) (*upload_v1.MediaReply, error) {
	media, err := s.uc.UploadMedia(ctx, req.FileName, req.Content)
	if err != nil {
		return nil, err
	}

	return &upload_v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *UploadService) UploadAvatar(ctx context.Context, req *upload_v1.UploadMediaRequest) (*upload_v1.MediaReply, error) {
	media, err := s.uc.UploadAvatar(ctx, req.FileName, req.Content)
	if err != nil {
		return nil, err
	}

	return &upload_v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *UploadService) GetMedia(ctx context.Context, req *upload_v1.GetMediaRequest) (*upload_v1.MediaReply, error) {
	media, err := s.uc.GetMedia(ctx, req.MediaId)
	if err != nil {
		return nil, err
	}

	return &upload_v1.MediaReply{Media: replyMedia(media)}, nil
}

func (s *UploadService) GetMediaList(ctx context.Context, req *upload_v1.GetMediaListRequest) (*upload_v1.MediaListReply, error) {
	mediaList, err := s.uc.GetMediaList(ctx, req.MediaIds)
	if err != nil {
		return nil, err
	}

	resultList := make([]*upload_v1.Media, len(mediaList))
	for i, media := range mediaList {
		resultList[i] = replyMedia(media)
	}

	return &upload_v1.MediaListReply{Media: resultList}, nil
}
