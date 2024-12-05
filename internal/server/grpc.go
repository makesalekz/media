package server

import (
	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/internal/conf"
	"gitlab.calendaria.team/services/media/internal/service"
	"gitlab.calendaria.team/services/utils/v1/middlewares/metrics"
	u_jwt "gitlab.calendaria.team/services/utils/v2/jwt"
	"gitlab.calendaria.team/services/utils/v2/middlewares/auth"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	ggrpc "google.golang.org/grpc"
)

const (
	maxRecvMsgSize = 2 * 10e9 // 2 GB
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Bootstrap, jwtp u_jwt.IJwtProcessor, srvc *service.MediaService) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			auth.Server(jwtp),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
				metrics.WithGauge(prom.NewGauge(_activeRequests)),
			),
		),
		grpc.Options(ggrpc.MaxRecvMsgSize(maxRecvMsgSize)),
	}
	if c.GetServer().GetGrpc().GetNetwork() != "" {
		opts = append(opts, grpc.Network(c.GetServer().GetGrpc().GetNetwork()))
	}
	if c.GetServer().GetGrpc().GetAddr() != "" {
		opts = append(opts, grpc.Address(c.GetServer().GetGrpc().GetAddr()))
	}
	if c.GetServer().GetGrpc().GetTimeout() != nil {
		opts = append(opts, grpc.Timeout(c.GetServer().GetGrpc().GetTimeout().AsDuration()))
	}
	srv := grpc.NewServer(opts...)

	v1.RegisterMediaServiceServer(srv, srvc)

	return srv
}
