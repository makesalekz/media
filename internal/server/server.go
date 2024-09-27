package server

import (
	"github.com/google/wire"
)

// ProviderSet is server providers.
//
//nolint:gochecknoglobals // global variable, used in wire
var ProviderSet = wire.NewSet(
	NewGRPCServer,
	NewHTTPServer,
)
