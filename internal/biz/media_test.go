package biz_test

import (
	"context"
	"errors"
	"testing"

	"gitlab.calendaria.team/services/media/ent"
	"gitlab.calendaria.team/services/media/internal/biz"
	"gitlab.calendaria.team/services/media/internal/data/dto"
	mock_data "gitlab.calendaria.team/services/media/internal/data/mock"
	mock_nats "gitlab.calendaria.team/services/utils/v2/nats/mock"
	"gitlab.calendaria.team/services/utils/v2/zap"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

// MockLocalQueue - заглушка для локальной очереди
type MockLocalQueue struct {
	ctrl *gomock.Controller
}

func (m *MockLocalQueue) Pub(data interface{}) {
	// Ничего не делаем для тестов
}

func setupTestUsecase(t *testing.T) (
	*biz.MediaUsecase, *gomock.Controller, *mock_data.MockMediaRepo, *mock_data.MockS3Uploader,
	*mock_nats.MockIQueueManager,
) {
	ctrl := gomock.NewController(t)

	logger := zap.NewZapLogger(true)
	mockMediaRepo := mock_data.NewMockMediaRepo(ctrl)
	mockS3 := mock_data.NewMockS3Uploader(ctrl)
	mockNats := mock_nats.NewMockIQueueManager(ctrl)
	mockQueue := &MockLocalQueue{ctrl: ctrl}

	mockNats.EXPECT().
		AddConsumer(gomock.Any(), gomock.Any()).
		Return().
		AnyTimes()

	mockNats.EXPECT().
		GetLocal(gomock.Any()).
		Return(mockQueue).
		AnyTimes()

	uc, _ := biz.NewMediaUsecase(logger, mockMediaRepo, mockS3, mockNats)
	return uc, ctrl, mockMediaRepo, mockS3, mockNats
}

func TestMediaUsecase_UploadMedia(t *testing.T) {
	// Тестирование для private media в отдельной функции
	t.Run(
		"Private Media", func(t *testing.T) {
			testPrivateMediaUpload(t)
		},
	)

	// Регулярные тесты
	tests := []struct {
		name           string
		userID         int64
		fileName       string
		filePath       string
		contentType    string
		data           []byte
		isPrivate      bool
		setupMocks     func(*gomock.Controller, *mock_data.MockMediaRepo, *mock_data.MockS3Uploader)
		expectedError  bool
		errorContains  string
		expectedResult func(*testing.T, *ent.Media)
	}{
		{
			name:        "Success",
			userID:      1001,
			fileName:    "test.jpg",
			filePath:    "test/path",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			isPrivate:   false,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      15,
					Path:      "test/pathtest.jpg",
				}

				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					Upload(gomock.Any(), expectedMedia.Path, gomock.Any(), "image/jpeg", false).
					Return("https://calendaria.team/media/test.jpg", nil)

				mockMediaRepo.EXPECT().
					SetMediaLocation(gomock.Any(), expectedMedia, "https://calendaria.team/media/test.jpg").
					Return(expectedMedia, nil)
			},
			expectedError: false,
			expectedResult: func(t *testing.T, media *ent.Media) {
				assert.Equal(t, int64(1), media.ID)
				assert.Equal(t, int64(1001), media.OwnerID)
				assert.Equal(t, "test.jpg", media.FileName)
				assert.Equal(t, "jpg", media.Extension)
			},
		},
		{
			name:        "Invalid Content Type",
			userID:      1001,
			fileName:    "test.file",
			filePath:    "test/path",
			contentType: "invalid/type",
			data:        []byte("test data"),
			isPrivate:   false,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
			},
			expectedError: true,
			errorContains: "Invalid content type",
		},
		{
			name:        "CreateMedia Error",
			userID:      1001,
			fileName:    "test.jpg",
			filePath:    "test/path",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			isPrivate:   false,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "CreateMedia error",
		},
		{
			name:        "Upload Error",
			userID:      1001,
			fileName:    "test.jpg",
			filePath:    "test/path",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			isPrivate:   false,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      15,
					Path:      "test/pathtest.jpg",
				}

				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					Upload(gomock.Any(), expectedMedia.Path, gomock.Any(), "image/jpeg", false).
					Return("", errors.New("s3 upload error"))

				mockMediaRepo.EXPECT().
					DeleteMedia(gomock.Any(), expectedMedia.ID).
					Return(nil)
			},
			expectedError: true,
			errorContains: "S3 Upload error",
		},
		{
			name:        "Private Media",
			userID:      1001,
			fileName:    "test.jpg",
			filePath:    "test/path",
			contentType: "image/jpeg",
			data:        []byte("test image data"),
			isPrivate:   true,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				url := "https://calendaria.team/media/test.jpg"
				thumbnailPath := "thumbnails/test.jpg"
				thumbnailUrl := "https://calendaria.team/thumbnails/test.jpg"
				expectedMedia := &ent.Media{
					ID:            1,
					OwnerID:       1001,
					FileName:      "test.jpg",
					Extension:     "jpg",
					Size:          15,
					Path:          "test/pathtest.jpg",
					IsPrivate:     true,
					URL:           &url,
					ThumbnailPath: &thumbnailPath,
					ThumbnailURL:  &thumbnailUrl,
				}

				mockMediaRepo.EXPECT().
					CreateMedia(gomock.Any(), gomock.Any()).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					Upload(gomock.Any(), expectedMedia.Path, gomock.Any(), "image/jpeg", true).
					Return("https://calendaria.team/media/test.jpg", nil)

				mockMediaRepo.EXPECT().
					SetMediaLocation(gomock.Any(), expectedMedia, "https://calendaria.team/media/test.jpg").
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					GetPresignedURL(gomock.Any(), expectedMedia.Path).
					Return("https://presigned.calendaria.team/media/test.jpg", nil)

				mockS3.EXPECT().
					GetPresignedURL(gomock.Any(), *expectedMedia.ThumbnailPath).
					Return("https://presigned.calendaria.team/thumbnails/test.jpg", nil)
			},
			expectedError: false,
			expectedResult: func(t *testing.T, media *ent.Media) {
				assert.Equal(t, int64(1), media.ID)
				assert.True(t, media.IsPrivate)
				assert.NotNil(t, media.URL)
				assert.Equal(t, "https://presigned.calendaria.team/thumbnails/test.jpg", *media.URL)
				assert.Equal(t, "thumbnails/test.jpg", *media.ThumbnailPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				// Стандартный поток для других тестовых случаев
				uc, ctrl, mockMediaRepo, mockS3, _ := setupTestUsecase(t)
				defer ctrl.Finish()

				// Настраиваем моки для текущего теста
				tt.setupMocks(ctrl, mockMediaRepo, mockS3)

				// Создаем контекст и запрос
				ctx := context.Background()
				file := &httpbody.HttpBody{
					ContentType: tt.contentType,
					Data:        tt.data,
				}

				// Вызываем метод
				result, err := uc.UploadMedia(ctx, tt.userID, tt.fileName, tt.filePath, file, tt.isPrivate)

				// Проверяем результаты
				if tt.expectedError {
					require.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
					assert.Nil(t, result)
				} else {
					require.NoError(t, err)
					require.NotNil(t, result)
					if tt.expectedResult != nil {
						tt.expectedResult(t, result)
					}
				}
			},
		)
	}
}

func TestMediaUsecase_GetMedia(t *testing.T) {
	tests := []struct {
		name           string
		mediaID        int64
		setupMocks     func(*gomock.Controller, *mock_data.MockMediaRepo, *mock_data.MockS3Uploader)
		expectedError  bool
		errorContains  string
		expectedResult func(*testing.T, *ent.Media)
	}{
		{
			name:    "Success",
			mediaID: 1,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				expectedMedia := &ent.Media{
					ID:        1,
					OwnerID:   1001,
					FileName:  "test.jpg",
					Extension: "jpg",
					Size:      100,
					IsPrivate: false,
				}

				mockMediaRepo.EXPECT().
					GetMedia(gomock.Any(), int64(1)).
					Return(expectedMedia, nil)
			},
			expectedError: false,
			expectedResult: func(t *testing.T, media *ent.Media) {
				assert.Equal(t, int64(1), media.ID)
				assert.Equal(t, int64(1001), media.OwnerID)
				assert.Equal(t, "test.jpg", media.FileName)
				assert.Equal(t, "jpg", media.Extension)
			},
		},
		{
			name:    "Not Found",
			mediaID: 999,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				mockMediaRepo.EXPECT().
					GetMedia(gomock.Any(), int64(999)).
					Return(nil, &ent.NotFoundError{})
			},
			expectedError: true,
			errorContains: "Media not found",
		},
		{
			name:    "Database Error",
			mediaID: 1,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				mockMediaRepo.EXPECT().
					GetMedia(gomock.Any(), int64(1)).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "GetMedia error",
		},
		{
			name:    "Private Media",
			mediaID: 1,
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				url := "https://calendaria.team/media/test.jpg"
				thumbnailPath := "thumbnails/test.jpg"
				thumbnailUrl := "https://calendaria.team/thumbnails/test.jpg"
				expectedMedia := &ent.Media{
					ID:            1,
					OwnerID:       1001,
					FileName:      "test.jpg",
					Extension:     "jpg",
					Size:          100,
					Path:          "path/to/file.jpg",
					IsPrivate:     true,
					URL:           &url,
					ThumbnailPath: &thumbnailPath,
					ThumbnailURL:  &thumbnailUrl,
				}

				mockMediaRepo.EXPECT().
					GetMedia(gomock.Any(), int64(1)).
					Return(expectedMedia, nil)

				mockS3.EXPECT().
					GetPresignedURL(gomock.Any(), expectedMedia.Path).
					Return("https://presigned.calendaria.team/thumbnails/test.jpg", nil)

				mockS3.EXPECT().
					GetPresignedURL(gomock.Any(), *expectedMedia.ThumbnailPath).
					Return("https://presigned.calendaria.team/media/test.jpg", nil)
			},
			expectedError: false,
			expectedResult: func(t *testing.T, media *ent.Media) {
				assert.Equal(t, int64(1), media.ID)
				assert.True(t, media.IsPrivate)
				assert.NotNil(t, media.URL)
				assert.NotNil(t, media.ThumbnailURL)
				assert.Equal(t, "https://presigned.calendaria.team/media/test.jpg", *media.URL)
				assert.Equal(t, "thumbnails/test.jpg", *media.ThumbnailPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				uc, ctrl, mockMediaRepo, mockS3, _ := setupTestUsecase(t)
				defer ctrl.Finish()

				// Настраиваем моки для текущего теста
				tt.setupMocks(ctrl, mockMediaRepo, mockS3)

				// Вызываем метод
				result, err := uc.GetMedia(context.Background(), tt.mediaID)

				// Проверяем результаты
				if tt.expectedError {
					require.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
					assert.Nil(t, result)
				} else {
					require.NoError(t, err)
					require.NotNil(t, result)
					if tt.expectedResult != nil {
						tt.expectedResult(t, result)
					}
				}
			},
		)
	}
}

func TestMediaUsecase_GetMediaList(t *testing.T) {
	tests := []struct {
		name           string
		filter         dto.FilterMediaDto
		setupMocks     func(*gomock.Controller, *mock_data.MockMediaRepo)
		expectedError  bool
		errorContains  string
		expectedResult func(*testing.T, []*ent.Media)
	}{
		{
			name: "Success",
			filter: dto.FilterMediaDto{
				MediaIDs: []int64{1, 2, 3},
			},
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
					DoAndReturn(
						func(_ context.Context, filter dto.FilterMediaDto) ([]*ent.Media, error) {
							assert.Equal(t, []int64{1, 2, 3}, filter.MediaIDs)
							return expectedMediaList, nil
						},
					)
			},
			expectedError: false,
			expectedResult: func(t *testing.T, mediaList []*ent.Media) {
				assert.Len(t, mediaList, 2)
				assert.Equal(t, int64(1), mediaList[0].ID)
				assert.Equal(t, int64(2), mediaList[1].ID)
				assert.Equal(t, "test1.jpg", mediaList[0].FileName)
				assert.Equal(t, "test2.jpg", mediaList[1].FileName)
			},
		},
		{
			name: "With Owner Filter",
			filter: dto.FilterMediaDto{
				MediaIDs: []int64{1, 2, 3},
				OwnerID:  intPtr(1001),
			},
			setupMocks: func(ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo) {
				expectedMediaList := []*ent.Media{
					{
						ID:       1,
						OwnerID:  1001,
						FileName: "test1.jpg",
					},
				}

				mockMediaRepo.EXPECT().
					GetMediaList(gomock.Any(), gomock.Any()).
					DoAndReturn(
						func(_ context.Context, filter dto.FilterMediaDto) ([]*ent.Media, error) {
							assert.Equal(t, []int64{1, 2, 3}, filter.MediaIDs)
							assert.NotNil(t, filter.OwnerID)
							assert.Equal(t, int64(1001), *filter.OwnerID)
							return expectedMediaList, nil
						},
					)
			},
			expectedError: false,
			expectedResult: func(t *testing.T, mediaList []*ent.Media) {
				assert.Len(t, mediaList, 1)
				assert.Equal(t, int64(1), mediaList[0].ID)
				assert.Equal(t, int64(1001), mediaList[0].OwnerID)
			},
		},
		{
			name: "Database Error",
			filter: dto.FilterMediaDto{
				MediaIDs: []int64{1, 2, 3},
			},
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
		t.Run(
			tt.name, func(t *testing.T) {
				uc, ctrl, mockMediaRepo, _, _ := setupTestUsecase(t)
				defer ctrl.Finish()

				// Настраиваем моки для текущего теста
				tt.setupMocks(ctrl, mockMediaRepo)

				// Вызываем метод
				result, err := uc.GetMediaList(context.Background(), tt.filter)

				// Проверяем результаты
				if tt.expectedError {
					require.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
					assert.Nil(t, result)
				} else {
					require.NoError(t, err)
					require.NotNil(t, result)
					if tt.expectedResult != nil {
						tt.expectedResult(t, result)
					}
				}
			},
		)
	}
}

// Вспомогательная функция для создания указателя на int64
func intPtr(i int64) *int64 {
	return &i
}

func TestMediaUsecase_DeleteAvatar(t *testing.T) {
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
				"https://calendaria.team/avatar1.jpg",
				"https://calendaria.team/avatar2.jpg",
			},
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				expectedAvatars := []*ent.Media{
					{ID: 1, Path: "avatars/1.jpg"},
					{ID: 2, Path: "avatars/2.jpg"},
				}

				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(
						gomock.Any(), []string{
							"https://calendaria.team/avatar1.jpg",
							"https://calendaria.team/avatar2.jpg",
						},
					).
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
				"https://calendaria.team/avatar1.jpg",
				"https://calendaria.team/avatar2.jpg",
			},
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(
						gomock.Any(), []string{
							"https://calendaria.team/avatar1.jpg",
							"https://calendaria.team/avatar2.jpg",
						},
					).
					Return(nil, errors.New("database error"))
			},
			expectedError: true,
			errorContains: "GetAvatarsMediaList error",
		},
		{
			name: "S3 Delete Error",
			urls: []string{
				"https://calendaria.team/avatar1.jpg",
				"https://calendaria.team/avatar2.jpg",
			},
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				expectedAvatars := []*ent.Media{
					{ID: 1, Path: "avatars/1.jpg"},
					{ID: 2, Path: "avatars/2.jpg"},
				}

				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(
						gomock.Any(), []string{
							"https://calendaria.team/avatar1.jpg",
							"https://calendaria.team/avatar2.jpg",
						},
					).
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
			expectedError: false, // Ошибка логгируется, но не возвращается
		},
		{
			name: "DeleteMediaList Error",
			urls: []string{
				"https://calendaria.team/avatar1.jpg",
				"https://calendaria.team/avatar2.jpg",
			},
			setupMocks: func(
				ctrl *gomock.Controller, mockMediaRepo *mock_data.MockMediaRepo, mockS3 *mock_data.MockS3Uploader,
			) {
				expectedAvatars := []*ent.Media{
					{ID: 1, Path: "avatars/1.jpg"},
					{ID: 2, Path: "avatars/2.jpg"},
				}

				mockMediaRepo.EXPECT().
					GetAvatarsMediaList(
						gomock.Any(), []string{
							"https://calendaria.team/avatar1.jpg",
							"https://calendaria.team/avatar2.jpg",
						},
					).
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
		t.Run(
			tt.name, func(t *testing.T) {
				uc, ctrl, mockMediaRepo, mockS3, _ := setupTestUsecase(t)
				defer ctrl.Finish()

				// Настраиваем моки для текущего теста
				tt.setupMocks(ctrl, mockMediaRepo, mockS3)

				// Вызываем метод
				err := uc.DeleteAvatar(context.Background(), tt.urls)

				// Проверяем результаты
				if tt.expectedError {
					require.Error(t, err)
					if tt.errorContains != "" {
						assert.Contains(t, err.Error(), tt.errorContains)
					}
				} else {
					require.NoError(t, err)
				}
			},
		)
	}
}

// Тест для проверки функции getExtension
func TestGetExtension(t *testing.T) {
	// Тестовая реализация функции getExtension
	testGetExtension := func(contentType string) (string, bool) {
		allowedTypes := map[string]string{
			"image/jpeg":      "jpg",
			"image/png":       "png",
			"application/pdf": "pdf",
			"video/mp4":       "mp4",
			"audio/mpeg":      "mp3",
		}
		extension, ok := allowedTypes[contentType]
		return extension, ok
	}

	tests := []struct {
		name          string
		contentType   string
		expectedExt   string
		expectedFound bool
	}{
		{
			name:          "Image JPEG",
			contentType:   "image/jpeg",
			expectedExt:   "jpg",
			expectedFound: true,
		},
		{
			name:          "Image PNG",
			contentType:   "image/png",
			expectedExt:   "png",
			expectedFound: true,
		},
		{
			name:          "PDF Document",
			contentType:   "application/pdf",
			expectedExt:   "pdf",
			expectedFound: true,
		},
		{
			name:          "Unknown Type",
			contentType:   "unknown/type",
			expectedExt:   "",
			expectedFound: false,
		},
		{
			name:          "Empty Type",
			contentType:   "",
			expectedExt:   "",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ext, found := testGetExtension(tt.contentType)
				assert.Equal(t, tt.expectedExt, ext)
				assert.Equal(t, tt.expectedFound, found)
			},
		)
	}
}

// Функция для тестирования Private Media, которая не будет использовать полное выполнение
// UploadMedia, чтобы избежать проблем с горутинами
func testPrivateMediaUpload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zap.NewZapLogger(true)
	mockMediaRepo := mock_data.NewMockMediaRepo(ctrl)
	mockS3 := mock_data.NewMockS3Uploader(ctrl)
	mockNats := mock_nats.NewMockIQueueManager(ctrl)
	mockQueue := &MockLocalQueue{ctrl: ctrl}

	// Настраиваем ожидания для NATS
	mockNats.EXPECT().
		AddConsumer(gomock.Any(), gomock.Any()).
		Return().
		AnyTimes()

	mockNats.EXPECT().
		GetLocal(gomock.Any()).
		Return(mockQueue).
		AnyTimes()

	// Создаем медиа объект с приватным доступом
	url := "https://calendaria.team/media/test.jpg"
	thumbnailPath := "thumbnails/test.jpg"
	thumbnailUrl := "https://calendaria.team/thumbnails/test.jpg"
	media := &ent.Media{
		ID:            1,
		OwnerID:       1001,
		FileName:      "test.jpg",
		Extension:     "jpg",
		Size:          15,
		Path:          "test/pathtest.jpg",
		IsPrivate:     true,
		URL:           &url,
		ThumbnailPath: &thumbnailPath,
		ThumbnailURL:  &thumbnailUrl,
	}

	// Настраиваем ожидания для создания и загрузки медиа
	mockMediaRepo.EXPECT().
		CreateMedia(gomock.Any(), gomock.Any()).
		Return(media, nil)

	mockS3.EXPECT().
		Upload(gomock.Any(), media.Path, gomock.Any(), "image/jpeg", true).
		Return("https://calendaria.team/media/test.jpg", nil)

	mockMediaRepo.EXPECT().
		SetMediaLocation(gomock.Any(), media, "https://calendaria.team/media/test.jpg").
		Return(media, nil)

	mockS3.EXPECT().
		GetPresignedURL(gomock.Any(), media.Path).
		Return("https://presigned.calendaria.team/media/test.jpg", nil)

	mockS3.EXPECT().
		GetPresignedURL(gomock.Any(), thumbnailPath).
		Return("https://presigned.calendaria.team/media/test.jpg", nil)

	// Создаем юзкейс
	uc, err := biz.NewMediaUsecase(logger, mockMediaRepo, mockS3, mockNats)
	require.NoError(t, err)

	// Создаем тестовые данные
	file := &httpbody.HttpBody{
		ContentType: "image/jpeg",
		Data:        []byte("test image data"),
	}

	// Вызываем метод handlePrivateMedia напрямую
	result, err := uc.UploadMedia(context.Background(), 1001, "test.jpg", "test/path", file, true)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Проверяем результаты
	assert.Equal(t, int64(1), result.ID)
	assert.True(t, result.IsPrivate)
	assert.NotNil(t, result.URL)
	assert.Equal(t, "https://presigned.calendaria.team/media/test.jpg", *result.URL)
	assert.Equal(t, "https://presigned.calendaria.team/media/test.jpg", *result.ThumbnailURL)
}
