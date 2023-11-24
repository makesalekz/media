package server

import (
	"io"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	jwtv4 "github.com/golang-jwt/jwt/v4"
	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/internal/conf"
	"gitlab.calendaria.team/services/media/internal/data"
	"gitlab.calendaria.team/services/media/internal/service"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Bootstrap, logger log.Logger, jwtp *data.JwtProcessor, srvc *service.MediaService) *khttp.Server {
	var opts = []khttp.ServerOption{
		khttp.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			jwt.Server(func(token *jwtv4.Token) (interface{}, error) {
				return jwtp.GetSecret(), nil
			}, jwt.WithSigningMethod(jwtv4.SigningMethodHS256), jwt.WithClaims(func() jwtv4.Claims { return &jwtv4.RegisteredClaims{} })),
		),
		khttp.RequestDecoder(func(r *http.Request, v interface{}) error {
			_, ok := khttp.CodecForRequest(r, "Content-Type")
			if ok {
				return khttp.DefaultRequestDecoder(r, v)
			}

			file, err := io.ReadAll(r.Body)
			if err != nil {
				return errors.BadRequest("CODEC", err.Error())
			}
			defer r.Body.Close()

			contentType := mimetype.Detect(file).String()
			if contentType == "application/octet-stream" && r.Header.Get("Content-Type") != "" {
				contentType = r.Header.Get("Content-Type")
			}

			v.(*v1.UploadMediaRequest).FileName = r.Header.Get("X-File-Name")
			v.(*v1.UploadMediaRequest).FilePath = r.Header.Get("X-File-Path")
			v.(*v1.UploadMediaRequest).Content = &httpbody.HttpBody{
				ContentType: contentType,
				Data:        file,
			}
			return nil
		}),
	}
	if c.Server.Http.Network != "" {
		opts = append(opts, khttp.Network(c.Server.Http.Network))
	}
	if c.Server.Http.Addr != "" {
		opts = append(opts, khttp.Address(c.Server.Http.Addr))
	}
	if c.Server.Http.Timeout != nil {
		opts = append(opts, khttp.Timeout(c.Server.Http.Timeout.AsDuration()))
	}
	srv := khttp.NewServer(opts...)

	v1.RegisterMediaServiceHTTPServer(srv, srvc)

	return srv
}
