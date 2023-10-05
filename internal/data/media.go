package data

import (
	"context"
	"fmt"
	"time"

	"media/ent"
	"media/ent/media"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// MediaRepo
type MediaRepo interface {
	CreateMedia(ctx context.Context, ownerId int64, fileName, extension string, size int) (*ent.Media, error)
	SetMediaLocation(ctx context.Context, media *ent.Media, location string) (*ent.Media, error)
	GetMedia(ctx context.Context, mediaId int64) (*ent.Media, error)
	GetMediaList(ctx context.Context, mediaIds []int64) ([]*ent.Media, error)
}

type mediaRepo struct {
	db *ent.Client
}

// NewMediaRepo .
func NewMediaRepo(d *Data) MediaRepo {
	return &mediaRepo{
		db: d.db,
	}
}

func (r *mediaRepo) CreateMedia(ctx context.Context, ownerId int64, fileName, extension string, size int) (*ent.Media, error) {
	uuid := uuid.NewString()
	path := fmt.Sprintf("%s/%s.%s", time.Now().Format("2006/01/02"), uuid, extension)

	return r.db.Media.Create().
		SetOwnerID(ownerId).
		SetFileName(fileName).
		SetPath(path).
		SetExtension(extension).
		SetSize(int32(size)).
		Save(ctx)
}

func (r *mediaRepo) SetMediaLocation(ctx context.Context, media *ent.Media, URL string) (*ent.Media, error) {
	return media.Update().
		SetURL(URL).
		SetUploadedAt(time.Now()).
		Save(ctx)
}

func (r *mediaRepo) GetMedia(ctx context.Context, mediaId int64) (*ent.Media, error) {
	return r.db.Media.Get(ctx, mediaId)
}

func (r *mediaRepo) GetMediaList(ctx context.Context, mediaIds []int64) ([]*ent.Media, error) {
	return r.db.Media.Query().Where(media.IDIn(mediaIds...)).All(ctx)
}
