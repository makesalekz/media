package data

type MediaHelper interface{}

type mediaHelper struct{}

func NewMediaHelper() MediaHelper {
	return &mediaHelper{}
}
