package service

import (
	"github.com/google/wire"
)

// ProviderSet is service providers.
//
//nolint:gochecknoglobals // global variable, used in wire
var ProviderSet = wire.NewSet(
	NewMediaService,
)
