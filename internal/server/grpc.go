package server

import (
	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	kjwt "github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/metrics"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	jwtv4 "github.com/golang-jwt/jwt/v4"
	media_v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/internal/conf"
	"gitlab.calendaria.team/services/media/internal/service"
	"gitlab.calendaria.team/services/utils/v1/jwt"
	ggrpc "google.golang.org/grpc"
)

const (
	maxRecvMsgSize = 100 * 10e6 // 100 MB
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Bootstrap, jwtp *jwt.JwtProcessor, srvc *service.MediaService) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			kjwt.Server(func(token *jwtv4.Token) (interface{}, error) {
				return jwtp.GetSecret(), nil
			}, kjwt.WithSigningMethod(jwtv4.SigningMethodHS256), kjwt.WithClaims(func() jwtv4.Claims { return &jwt.TenantClaims{} })),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
			),
		),
		grpc.Options(ggrpc.MaxRecvMsgSize(maxRecvMsgSize)),
	}
	if c.Server.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Server.Grpc.Network))
	}
	if c.Server.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Server.Grpc.Addr))
	}
	if c.Server.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Server.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)

	media_v1.RegisterMediaServiceServer(srv, srvc)

	return srv
}
