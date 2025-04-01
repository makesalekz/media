package dto

type CreateMediaDto struct {
	OwnerID   int64
	FileName  string
	Path      string
	Extension string
	Size      int32
	IsPrivate bool
}

type SetVideoParamsDto struct {
	Duration      float32
	ThumbnailURL  string
	ThumbnailPath string
}

type SetDimensionsDto struct {
	Width  int32
	Height int32
}

type FilterMediaDto struct {
	OwnerID  *int64
	MediaIDs []int64
}
