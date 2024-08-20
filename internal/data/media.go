package data

import (
	"context"
	"gitlab.calendaria.team/services/media/ent/mixins"
	"time"

	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/ent/media"

	_ "github.com/lib/pq"
)

type CreateMediaDto struct {
	OwnerId   int64
	FileName  string
	Path      string
	Extension string
	Size      int32
}

type SetVideoParamsDto struct {
	Duration      float32
	ThumbnailUrl  string
	ThumbnailPath string
}

type SetDimensionsDto struct {
	Width  int32
	Height int32
}

type FilterMediaDto struct {
	OwnerId  *int64
	MediaIds []int64
}

// MediaRepo
type MediaRepo interface {
	CreateMedia(ctx context.Context, dto CreateMediaDto) (*ent.Media, error)
	DeleteMedia(ctx context.Context, mediaId int64) error
	SetMediaLocation(ctx context.Context, media *ent.Media, location string) (*ent.Media, error)
	GetMedia(ctx context.Context, mediaId int64) (*ent.Media, error)
	SetVideoParameters(ctx context.Context, video *ent.Media, dto SetVideoParamsDto) (*ent.Media, error)
	SetDimensions(ctx context.Context, media *ent.Media, dto SetDimensionsDto) (*ent.Media, error)
	GetMediaList(ctx context.Context, filter FilterMediaDto) ([]*ent.Media, error)
	GetAvatarsMediaList(ctx context.Context, urls []string) ([]*ent.Media, error)
	DeleteMediaList(ctx context.Context, ids []int64) (int, error)
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

func (r *mediaRepo) CreateMedia(ctx context.Context, dto CreateMediaDto) (*ent.Media, error) {
	return r.db.Media.Create().
		SetOwnerID(dto.OwnerId).
		SetFileName(dto.FileName).
		SetPath(dto.Path).
		SetExtension(dto.Extension).
		SetSize(dto.Size).
		Save(ctx)
}

func (r *mediaRepo) DeleteMedia(ctx context.Context, mediaId int64) error {
	_, err := r.db.Media.Delete().Where(media.ID(mediaId)).Exec(ctx)
	return err
}

func (r *mediaRepo) SetMediaLocation(ctx context.Context, media *ent.Media, URL string) (*ent.Media, error) {
	return media.Update().
		SetURL(URL).
		SetUploadedAt(time.Now()).
		Save(ctx)
}

func (r *mediaRepo) SetVideoParameters(ctx context.Context, video *ent.Media, dto SetVideoParamsDto) (*ent.Media, error) {
	return video.Update().
		SetThumbnailURL(dto.ThumbnailUrl).
		SetThumbnailPath(dto.ThumbnailPath).
		SetDuration(dto.Duration).
		Save(ctx)
}

func (r *mediaRepo) SetDimensions(ctx context.Context, media *ent.Media, dto SetDimensionsDto) (*ent.Media, error) {
	return media.Update().
		SetHeight(dto.Height).
		SetWidth(dto.Width).
		Save(ctx)
}

func (r *mediaRepo) GetMedia(ctx context.Context, mediaId int64) (*ent.Media, error) {
	return r.db.Media.Get(ctx, mediaId)
}

func (r *mediaRepo) GetMediaList(ctx context.Context, filter FilterMediaDto) ([]*ent.Media, error) {
	query := r.db.Media.Query().Where(media.IDIn(filter.MediaIds...))

	if filter.OwnerId != nil {
		query.Where(media.OwnerID(*filter.OwnerId))
	}

	return query.All(ctx)
}

func (r *mediaRepo) GetAvatarsMediaList(ctx context.Context, urls []string) ([]*ent.Media, error) {
	return r.db.Media.Query().Where(media.URLIn(urls...)).All(ctx)
}

func (r *mediaRepo) DeleteMediaList(ctx context.Context, ids []int64) (int, error) {
	return r.db.Media.Delete().Where(media.IDIn(ids...)).Exec(mixins.SkipSoftDelete(ctx))
}
