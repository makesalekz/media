package biz

import "github.com/google/wire"

const QueueDeleteMedia = "delete"

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewQueueManager, NewMediaUsecase)
