package biz

import (
	"context"
	_ "embed"
	"encoding/json"
	upload_v1 "media/api/upload/v1"
	"media/ent"
	"strings"

	"media/internal/conf"
	"media/internal/data"

	users "media/third_party/api/users/v1"

	consul "github.com/go-kratos/consul/registry"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	jwtv4 "github.com/golang-jwt/jwt/v4"
	"github.com/nats-io/nats.go"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// MediaUsecase is a Greeter usecase.
type MediaUsecase struct {
	conf      *conf.Bootstrap
	log       *log.Helper
	discovery *consul.Registry
	jwt       *data.JwtProcessor
	mediaRepo data.MediaRepo
	s3        *data.S3Uploader
	qm        *QueueManager
}

// NewGreeterUsecase new a Greeter usecase.
func NewMediaUsecase(
	logger log.Logger,
	c *data.Config,
	jwt *data.JwtProcessor,
	mediaRepo data.MediaRepo,
	s3 *data.S3Uploader,
	qm *QueueManager,
) (*MediaUsecase, error) {
	uc := &MediaUsecase{
		conf:      c.Bootstrap,
		log:       log.NewHelper(logger),
		discovery: c.GetRegistry(),
		jwt:       jwt,
		mediaRepo: mediaRepo,
		s3:        s3,
		qm:        qm,
	}

	qm.AddConsumer(QueueDeleteMedia, uc.deleteMediaConsumer)

	return uc, nil
}

func (uc *MediaUsecase) deleteMediaConsumer(ctx context.Context, m *nats.Msg) bool {
	var mediaId int64
	err := json.Unmarshal(m.Data, &mediaId)
	if err != nil {
		uc.log.Errorf("deleteMediaConsumer: json.Unmarshal: %s", err.Error())
		return true
	}

	media, err := uc.mediaRepo.GetMedia(ctx, mediaId)
	if err != nil {
		return false
	}

	err = uc.s3.Delete(ctx, media.Path)
	if err != nil {
		return false
	}

	err = uc.mediaRepo.DeleteMedia(ctx, media.ID)
	if err != nil {
		return true
	}

	return true
}

func getExtension(contentType string) (string, bool) {
	allowedContentTypes := []string{
		"image/jpeg",
		"image/png",
		"image/webp",
	}

	for _, allowedContentType := range allowedContentTypes {
		if contentType == allowedContentType {
			return strings.Split(contentType, "/")[1], true
		}
	}

	return "", false
}

func (uc *MediaUsecase) dialIam(ctx context.Context) (users.UsersClient, error) {
	conn, err := grpc.DialInsecure(
		ctx,
		grpc.WithEndpoint(uc.conf.Discovery.Iam),
		grpc.WithDiscovery(uc.discovery),
		grpc.WithTimeout(uc.conf.Discovery.IamTimeout.AsDuration()),
		grpc.WithMiddleware(
			jwt.Client(func(token *jwtv4.Token) (interface{}, error) {
				return uc.jwt.GetSecret(), nil
			}, jwt.WithSigningMethod(jwtv4.SigningMethodHS256), jwt.WithClaims(func() jwtv4.Claims {
				return uc.jwt.GetClaimsFromContext(ctx)
			})),
		),
	)
	if err != nil {
		return nil, err
	}
	return users.NewUsersClient(conn), nil
}

func (uc *MediaUsecase) UploadMedia(ctx context.Context, fileName string, file *httpbody.HttpBody) (*ent.Media, error) {
	userId, ok := uc.jwt.GetUserIdFromContext(ctx)
	if !ok {
		return nil, upload_v1.ErrorUnauthorized("Unauthorized")
	}

	contentType := file.GetContentType()
	extension, ok := getExtension(contentType)
	if !ok {
		return nil, upload_v1.ErrorInvalidContentType("Invalid content type: %s", contentType)
	}

	media, err := uc.mediaRepo.CreateMedia(ctx, userId, fileName, extension, len(file.GetData()))
	if err != nil {
		return nil, upload_v1.ErrorDatabaseQuery("CreateMedia error: %s", err)
	}

	location, err := uc.s3.Upload(ctx, media.Path, file)
	if err != nil {
		uc.mediaRepo.DeleteMedia(ctx, media.ID)

		return nil, upload_v1.ErrorS3uploadFailed("S3 Upload error: %s", err)
	}

	media, err = uc.mediaRepo.SetMediaLocation(ctx, media, location)
	if err != nil {
		return nil, upload_v1.ErrorDatabaseQuery("SetMediaUploadedAt error: %s", err)
	}

	return media, nil
}

func (uc *MediaUsecase) UploadAvatar(ctx context.Context, fileName string, file *httpbody.HttpBody) (*ent.Media, error) {
	media, err := uc.UploadMedia(ctx, fileName, file)
	if err != nil {
		return nil, err
	}

	usersClient, err := uc.dialIam(ctx)
	if err != nil {
		return nil, upload_v1.ErrorGrpcConnection("dialIam: %s", err.Error())
	}

	reply, err := usersClient.UpdateOwnProfile(ctx, &users.UpdateOwnProfileRequest{
		Avatar: *media.URL,
	})
	if err != nil {
		return nil, upload_v1.ErrorServiceFailed("users.UpdateOwnProfile: %s", err.Error())
	}
	uc.log.Infof("users.UpdateOwnProfile avatar: %s", reply.User.GetAvatar())

	return media, nil
}

func (uc *MediaUsecase) GetMedia(ctx context.Context, mediaId int64) (*ent.Media, error) {
	media, err := uc.mediaRepo.GetMedia(ctx, mediaId)
	if err != nil {
		return nil, upload_v1.ErrorDatabaseQuery("GetMedia error: %s", err)
	}

	return media, nil
}

func (uc *MediaUsecase) GetMediaList(ctx context.Context, mediaIds []int64) ([]*ent.Media, error) {
	mediaList, err := uc.mediaRepo.GetMediaList(ctx, mediaIds)
	if err != nil {
		return nil, upload_v1.ErrorDatabaseQuery("GetMediaList error: %s", err)
	}

	return mediaList, nil
}
