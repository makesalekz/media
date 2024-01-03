package server

import (
	"io"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/internal/conf"
	"gitlab.calendaria.team/services/utils/v1/jwt"
	auth "gitlab.calendaria.team/services/utils/v1/middlewares/auth"
	metrics "gitlab.calendaria.team/services/utils/v1/middlewares/metrics"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

var _metricSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "duration_sec",
	Help:      "server requests duratio(sec).",
	Buckets:   []float64{0.250, 0.5, 1, 5, 10, 30, 60},
}, []string{"kind", "operation"})

var _metricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "code_total",
	Help:      "The total number of processed requests",
}, []string{"kind", "operation", "code", "reason"})

var _activeRequests = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "server",
	Subsystem: "requests",
	Name:      "active_requests",
	Help:      "The total number of active requests",
}, []string{"kind", "operation"})

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Bootstrap, jwtp *jwt.JwtProcessor) *khttp.Server {
	var opts = []khttp.ServerOption{
		khttp.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			auth.Server(jwtp),
			metrics.Server(
				metrics.WithSeconds(prom.NewHistogram(_metricSeconds)),
				metrics.WithRequests(prom.NewCounter(_metricRequests)),
				metrics.WithGauge(prom.NewGauge(_activeRequests)),
			),
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

	registerTechRoutes(srv)

	return srv
}

func registerTechRoutes(s *khttp.Server) {
	prometheus.MustRegister(_metricSeconds, _metricRequests)

	s.Handle("/metrics", promhttp.Handler())
}
