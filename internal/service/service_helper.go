package service

import (
	"context"

	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/utils/v1/jwt"
)

type ServiceHelper struct {
	jwt *jwt.JwtProcessor
}

func NewServiceHelper(
	jwt *jwt.JwtProcessor,
) *ServiceHelper {
	return &ServiceHelper{
		jwt: jwt,
	}
}

func (s *ServiceHelper) GetActorId(ctx context.Context, reqActorId int64) (int64, error) {
	// TODO: remove getting from context
	actorId := s.jwt.GetUserIdFromContext(ctx)
	if actorId != 0 {
		return actorId, nil
	}

	if reqActorId != 0 {
		return reqActorId, nil
	}
	return 0, v1.ErrorInvalidRequest("empty actor id")
}
