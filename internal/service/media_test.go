package service_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	v1 "gitlab.calendaria.team/services/media/api/media/v1"
	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/biz"
	"gitlab.calendaria.team/services/media/internal/data/dto"
	mock_data "gitlab.calendaria.team/services/media/internal/data/mock"
	"gitlab.calendaria.team/services/media/internal/service"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
	mock_nats "gitlab.calendaria.team/services/utils/v2/nats/mock"
	"gitlab.calendaria.team/services/utils/v2/zap"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

func appendTestContext(appID string, actorID int64, tenantID int64) context.Context {
	ctx := context.Background()
	md := map[string][]string{}

	if appID != "" {
		md["x-md-global-app-id"] = []string{appID}
	}

	if actorID != 0 {
		md["x-md-global-actor-id"] = []string{
			strconv.FormatInt(actorID, 10),
		}
	}

	if tenantID != 0 {
		md["x-md-global-tenant-id"] = []string{
			strconv.FormatInt(tenantID, 10),
		}
	}

	return metadata.NewServerContext(ctx, metadata.New(md))
}

func setupTestService(t *testing.T) (
	*service.MediaService, *gomock.Controller, *mock_data.MockMediaRepo, *mock_data.MockS3Uploader,
) {
	ctrl := gomock.NewController(t)

	logger := zap.NewZapLogger(true)
	mockMediaRepo := mock_data.NewMockMediaRepo(ctrl)
	mockS3 := mock_data.NewMockS3Uploader(ctrl)
	mockNats := mock_nats.NewMockIQueueManager(ctrl)

	mockNats.EXPECT().
		AddConsumer(gomock.Any(), gomock.Any()).
		Return().
		AnyTimes()

	uc, _ := biz.NewMediaUsecase(logger, mockMediaRepo, mockS3, mockNats)

	svc := service.NewMediaService(uc)
	return svc, ctrl, mockMediaRepo, mockS3
}

func TestMediaService_UploadMedia(t *testing.T) {
	tests := []struct {
		name            string
		actorID         int64
		appID           string
		fileName        string
		filePath        string
		contentType     string
		data            []byte
		isPrivate       bool
		setupMocks      func(*gomock.Controller, *mock_data.MockMediaRepo, *mock_data.MockS3Uploader)
		expectedError   bool
		errorContains   string
		expectedMediaID int64
	}{
		{
			name:        "Success",
			actorID:     1001,
			appID:       "test-app",
			fileName:    "test.jpg",
			filePath:    "test/path",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			isPrivate:   false,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      100,
					Path:      "test/pathtest.jpg",
				}

				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					Upload(gomock.Any(), expectedMedia.Path, gomock.Any(), "image/jpeg", false).
					Return("https://example.com/media/test.jpg", nil)

				mockMediaRepo.EXPECT().
					SetMediaLocation(gomock.Any(), expectedMedia, "https://example.com/media/test.jpg").
					Return(expectedMedia, nil)
			},
			expectedError:   false,
			expectedMediaID: 1,
		},
		{
			name:        "Empty Actor ID",
			actorID:     0,
			appID:       "test-app",
			fileName:    "test.jpg",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
			},
			expectedError: true,
			errorContains: "empty actor id",
		},
		{
			name:        "Empty App ID",
			actorID:     1001,
			appID:       "",
			fileName:    "test.jpg",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
			},
			expectedError: true,
			errorContains: "empty app id",
		},
		{
			name:        "Invalid FileName",
			actorID:     1001,
			appID:       "test-app",
			fileName:    "%нек%%орректное%имя%",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
			},
			expectedError: true,
			errorContains: "invalid file name",
		},
		{
			name:        "CreateMedia Error",
			actorID:     1001,
			appID:       "test-app",
			fileName:    "test.jpg",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "CreateMedia error",
		},
		{
			name:        "Upload Error",
			actorID:     1001,
			appID:       "test-app",
			fileName:    "test.jpg",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      100,
					Path:      "path/to/file.jpg",
				}

				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return("", errors.New("s3 upload error"))

				mockMediaRepo.EXPECT().
					DeleteMedia(gomock.Any(), expectedMedia.ID).
					Return(nil)
			},
			expectedError: true,
			errorContains: "S3 Upload error",
		},
		{
			name:        "SetMediaLocation Error",
			actorID:     1001,
			appID:       "test-app",
			fileName:    "test.jpg",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      100,
					Path:      "path/to/file.jpg",
				}

				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return("https://example.com/media/test.jpg", nil)

				mockMediaRepo.EXPECT().
					SetMediaLocation(gomock.Any(), expectedMedia, "https://example.com/media/test.jpg").
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "SetMediaUploadedAt error",
		},
		{
			name:        "Private Media",
			actorID:     1001,
			appID:       "test-app",
			fileName:    "test.jpg",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			isPrivate:   true,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				url := "https://example.com/media/test.jpg"
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      100,
					Path:      "path/to/file.jpg",
					IsPrivate: true,
					URL:       &url,
				}

				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					Upload(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), true).
					Return("https://example.com/media/test.jpg", nil)

				mockMediaRepo.EXPECT().
					SetMediaLocation(gomock.Any(), expectedMedia, "https://example.com/media/test.jpg").
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					GetPresignedURL(gomock.Any(), expectedMedia.Path).
					Return("https://presigned.example.com/media/test.jpg", nil)
			},
			expectedError:   false,
			expectedMediaID: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctrl, mockMediaRepo, mockS3 := setupTestService(t)
			defer ctrl.Finish()

			// Настраиваем моки для текущего теста
			tt.setupMocks(ctrl, mockMediaRepo, mockS3)

			// Создаем контекст и запрос
			ctx := appendTestContext(tt.appID, tt.actorID, 0)
			req := &v1.UploadMediaRequest{
				FileName: tt.fileName,
				FilePath: tt.filePath,
				Content: &httpbody.HttpBody{
					ContentType: tt.contentType,
					Data:        tt.data,
				},
				IsPrivate: tt.isPrivate,
			}

			// Вызываем метод
			resp, err := svc.UploadMedia(ctx, req)

			// Проверяем результаты
			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.expectedMediaID, resp.Media.Id)
			}
		})
	}
}

func TestMediaService_GetMedia(t *testing.T) {
	tests := []struct {
		name          string
		mediaID       int64
		setupMocks    func(*gomock.Controller, *mock_data.MockMediaRepo)
		expectedError bool
		errorContains string
		checkResponse func(*testing.T, *v1.MediaReply)
	}{
		{
			name:    "Success",
			mediaID: 1,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo) {
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      100,
				}

				mockMediaRepo.EXPECT().
					GetMedia(gomock.Any(), int64(1)).
					Return(expectedMedia, nil)
			},
			expectedError: false,
			checkResponse: func(t *testing.T, resp *v1.MediaReply) {
				require.NotNil(t, resp)
				assert.Equal(t, int64(1), resp.Media.Id)
				assert.Equal(t, int64(1001), resp.Media.OwnerId)
				assert.Equal(t, "test.jpg", resp.Media.FileName)
				assert.Equal(t, "jpg", resp.Media.Extension)
			},
		},
		{
			name:    "Not Found",
			mediaID: 999,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo) {
				mockMediaRepo.EXPECT().
					GetMedia(gomock.Any(), int64(999)).
					Return(nil, errors.New("media not found"))
			},
			expectedError: true,
			errorContains: "media not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctrl, mockMediaRepo, _ := setupTestService(t)
			defer ctrl.Finish()

			// Настраиваем моки для текущего теста
			tt.setupMocks(ctrl, mockMediaRepo)

			// Создаем запрос
			req := &v1.GetMediaRequest{
				MediaId: tt.mediaID,
			}

			// Вызываем метод
			resp, err := svc.GetMedia(context.Background(), req)

			// Проверяем результаты
			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestMediaService_GetMediaList(t *testing.T) {
	tests := []struct {
		name          string
		ctx           context.Context
		mediaIDs      []int64
		ownOnly       bool
		setupMocks    func(*gomock.Controller, *mock_data.MockMediaRepo)
		expectedError bool
		errorContains string
		checkResponse func(*testing.T, *v1.MediaListReply)
	}{
		{
			name:     "Success With Owner Filter",
			ctx:      appendTestContext("test-app", 1001, 0),
			mediaIDs: []int64{1, 2, 3},
			ownOnly:  true,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo) {
				expectedMediaList := []*ent.Media{
					{
						ID:       1,
						OwnerID:  1001,
						FileName: "test1.jpg",
					},
					{
						ID:       2,
						OwnerID:  1001,
						FileName: "test2.jpg",
					},
				}

				mockMediaRepo.EXPECT().
					GetMediaList(gomock.Any(), gomock.Any()).
					Do(
						func(_ context.Context, filter dto.FilterMediaDto) {
							assert.Equal(t, []int64{1, 2, 3}, filter.MediaIDs)
							assert.NotNil(t, filter.OwnerID)
							assert.Equal(t, int64(1001), *filter.OwnerID)
						},
					).
					Return(expectedMediaList, nil)
			},
			expectedError: false,
			checkResponse: func(t *testing.T, resp *v1.MediaListReply) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Media, 2)
				assert.Equal(t, int64(1), resp.Media[0].Id)
				assert.Equal(t, int64(2), resp.Media[1].Id)
				assert.Equal(t, "test1.jpg", resp.Media[0].FileName)
				assert.Equal(t, "test2.jpg", resp.Media[1].FileName)
			},
		},
		{
			name:     "Empty Actor ID With OwnOnly",
			ctx:      appendTestContext("test-app", 0, 0),
			mediaIDs: []int64{1, 2, 3},
			ownOnly:  true,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo) {
				// Не ожидаем вызова GetMediaList
			},
			expectedError: true,
			errorContains: "empty actor id",
		},
		{
			name:     "No Filter",
			ctx:      context.Background(),
			mediaIDs: []int64{1, 2, 3},
			ownOnly:  false,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo) {
				expectedMediaList := []*ent.Media{
					{
						ID:       1,
						OwnerID:  1001,
						FileName: "test1.jpg",
					},
					{
						ID:       2,
						OwnerID:  1002,
						FileName: "test2.jpg",
					},
				}

				mockMediaRepo.EXPECT().
					GetMediaList(gomock.Any(), gomock.Any()).
					Do(
						func(_ context.Context, filter dto.FilterMediaDto) {
							assert.Equal(t, []int64{1, 2, 3}, filter.MediaIDs)
							assert.Nil(t, filter.OwnerID)
						},
					).
					Return(expectedMediaList, nil)
			},
			expectedError: false,
			checkResponse: func(t *testing.T, resp *v1.MediaListReply) {
				require.NotNil(t, resp)
				assert.Len(t, resp.Media, 2)
			},
		},
		{
			name:     "Database Error",
			ctx:      context.Background(),
			mediaIDs: []int64{1, 2, 3},
			ownOnly:  false,
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo) {
				mockMediaRepo.EXPECT().
					GetMediaList(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "GetMediaList error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctrl, mockMediaRepo, _ := setupTestService(t)
			defer ctrl.Finish()

			// Настраиваем моки для текущего теста
			tt.setupMocks(ctrl, mockMediaRepo)

			// Создаем запрос
			req := &v1.GetMediaListRequest{
				MediaIds: tt.mediaIDs,
				OwnOnly:  tt.ownOnly,
			}

			// Вызываем метод
			resp, err := svc.GetMediaList(tt.ctx, req)

			// Проверяем результаты
			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}
		})
	}
}

func TestMediaService_DeleteAvatar(t *testing.T) {
	tests := []struct {
		name          string
		urls          []string
		setupMocks    func(*gomock.Controller, *mock_data.MockMediaRepo, *mock_data.MockS3Uploader)
		expectedError bool
		errorContains string
	}{
		{
			name: "Success",
			urls: []string{
				"https://example.com/avatar1.jpg",
				"https://example.com/avatar2.jpg",
			},
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				expectedAvatars := []*ent.Media{
					{ID: 1, Path: "avatars/1.jpg"},
					{ID: 2, Path: "avatars/2.jpg"},
				}

				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(gomock.Any(), gomock.Any()).
					Return(expectedAvatars, nil)

				mockS3.EXPECT().
					Delete(gomock.Any(), "avatars/1.jpg").
					Return(nil)

				mockS3.EXPECT().
					Delete(gomock.Any(), "avatars/2.jpg").
					Return(nil)

				mockMediaRepo.EXPECT().
					DeleteMediaList(gomock.Any(), []int64{1, 2}).
					Return(2, nil)
			},
			expectedError: false,
		},
		{
			name: "GetAvatarsMediaList Error",
			urls: []string{
				"https://example.com/avatar1.jpg",
				"https://example.com/avatar2.jpg",
			},
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "GetAvatarsMediaList error",
		},
		{
			name: "S3 Delete Error",
			urls: []string{
				"https://example.com/avatar1.jpg",
				"https://example.com/avatar2.jpg",
			},
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				expectedAvatars := []*ent.Media{
					{ID: 1, Path: "avatars/1.jpg"},
					{ID: 2, Path: "avatars/2.jpg"},
				}

				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(gomock.Any(), gomock.Any()).
					Return(expectedAvatars, nil)

				mockS3.EXPECT().
					Delete(gomock.Any(), "avatars/1.jpg").
					Return(errors.New("s3 delete error"))

				mockS3.EXPECT().
					Delete(gomock.Any(), "avatars/2.jpg").
					Return(nil)

				mockMediaRepo.EXPECT().
					DeleteMediaList(gomock.Any(), []int64{1, 2}).
					Return(2, nil)
			},
			expectedError: false, // Даже если ошибка в S3, мы продолжаем выполнение
		},
		{
			name: "DeleteMediaList Error",
			urls: []string{
				"https://example.com/avatar1.jpg",
				"https://example.com/avatar2.jpg",
			},
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader) {
				expectedAvatars := []*ent.Media{
					{ID: 1, Path: "avatars/1.jpg"},
					{ID: 2, Path: "avatars/2.jpg"},
				}

				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(gomock.Any(), gomock.Any()).
					Return(expectedAvatars, nil)

				mockS3.EXPECT().
					Delete(gomock.Any(), "avatars/1.jpg").
					Return(nil)

				mockS3.EXPECT().
					Delete(gomock.Any(), "avatars/2.jpg").
					Return(nil)

				mockMediaRepo.EXPECT().
					DeleteMediaList(gomock.Any(), []int64{1, 2}).
					Return(0, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "DeleteMediaList error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctrl, mockMediaRepo, mockS3 := setupTestService(t)
			defer ctrl.Finish()

			// Настраиваем моки для текущего теста
			tt.setupMocks(ctrl, mockMediaRepo, mockS3)

			// Создаем запрос
			req := &v1.DeleteAvatarRequest{
				Urls: tt.urls,
			}

			// Вызываем метод
			resp, err := svc.DeleteAvatar(context.Background(), req)

			// Проверяем результаты
			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.IsType(t, &utils_v1.EmptyReply{}, resp)
			}
		})
	}
}
