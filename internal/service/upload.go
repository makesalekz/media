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
	trMedia := &upload_v1.Media{
		Id:        media.ID,
		FileName:  media.FileName,
		Extension: media.Extension,
		Size:      media.Size,
		Width:     media.Width,
		Height:    media.Height,
	}
	if media.Location != nil {
		trMedia.Url = *media.Location
	}

	return trMedia
}

func (s *UploadService) UploadMedia(ctx context.Context, req *upload_v1.UploadMediaRequest) (*upload_v1.UploadMediaReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, upload_v1.ErrorUnauthorized("Unauthorized")
	}

	media, err := s.uc.UploadMedia(ctx, userId, req.FileName, req.Content)
	if err != nil {
		return nil, err
	}

	return &upload_v1.UploadMediaReply{Media: replyMedia(media)}, nil
}

func (s *UploadService) UploadAvatar(ctx context.Context, req *upload_v1.UploadMediaRequest) (*upload_v1.UploadMediaReply, error) {
	userId, ok := s.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, upload_v1.ErrorUnauthorized("Unauthorized")
	}

	media, err := s.uc.UploadAvatar(ctx, userId, req.FileName, req.Content)
	if err != nil {
		return nil, err
	}

	return &upload_v1.UploadMediaReply{Media: replyMedia(media)}, nil
}
