package data

import (
	"context"
	"fmt"
	"time"

	"media/ent"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// MediaRepo
type MediaRepo interface {
	CreateMedia(ctx context.Context, userId int64, fileName, extension string, size int) (*ent.Media, error)
	SetMediaLocation(ctx context.Context, media *ent.Media, location string) (*ent.Media, error)
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

func (r *mediaRepo) CreateMedia(ctx context.Context, userId int64, fileName, extension string, size int) (*ent.Media, error) {
	uuid := uuid.NewString()
	path := fmt.Sprintf("%s/%s.%s", time.Now().Format("2006/01/02"), uuid, extension)

	return r.db.Media.Create().
		SetUserID(userId).
		SetFileName(fileName).
		SetPath(path).
		SetExtension(extension).
		SetSize(int32(size)).
		Save(ctx)
}

func (r *mediaRepo) SetMediaLocation(ctx context.Context, media *ent.Media, location string) (*ent.Media, error) {
	return media.Update().
		SetLocation(location).
		SetUploadedAt(time.Now()).
		Save(ctx)
}
