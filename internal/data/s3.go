package data

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"gitlab.calendaria.team/services/utils/v1/config"
)

type S3Uploader struct {
	Bucket  string
	Session *session.Session
}

// NewS3Uploader .
func NewS3Uploader(c *config.Config) (*S3Uploader, error) {
	uploader := &S3Uploader{}

	if os.Getenv("DEBUG") == "" {
		region, err := c.Value("AWS_REGION").String()
		if err != nil {
			return nil, fmt.Errorf("failed to get AWS_REGION: %s", err.Error())
		}
		bucket, err := c.Value("AWS_BUCKET").String()
		if err != nil {
			return nil, fmt.Errorf("failed to get AWS_BUCKET: %s", err.Error())
		}

		sess, err := session.NewSession(
			&aws.Config{
				Region: aws.String(region),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("AWS Session error: %s", err.Error())
		}

		uploader = &S3Uploader{
			Bucket:  bucket,
			Session: sess,
		}
	}

	return uploader, nil
}

func (u *S3Uploader) Upload(ctx context.Context, path string, fileData []byte, mimeType string, isPrivate bool) (
	string, error,
) {
	if os.Getenv("DEBUG") != "" {
		return path + "_debug", nil
	}

	s3Request := &s3manager.UploadInput{
		Bucket:      aws.String(u.Bucket),
		Key:         aws.String(path),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(mimeType),
	}

	if !isPrivate {
		s3Request.ACL = aws.String("public-read")
	}

	uploader := s3manager.NewUploader(u.Session)
	out, err := uploader.Upload(s3Request)
	if err != nil {
		return "", err
	}

	return out.Location, nil
}

func (u *S3Uploader) GetPresignedURL(ctx context.Context, path string) (string, error) {
	if os.Getenv("DEBUG") != "" {
		return path + "_debug", nil
	}

	svc := s3.New(u.Session)

	req, _ := svc.GetObjectRequest(
		&s3.GetObjectInput{
			Bucket: aws.String(u.Bucket),
			Key:    aws.String(path),
		},
	)

	url, err := req.Presign(DefaultTimeout)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (u *S3Uploader) Delete(ctx context.Context, path string) error {
	if os.Getenv("DEBUG") != "" {
		return nil
	}

	batcher := s3manager.NewBatchDelete(u.Session)
	return batcher.Delete(
		aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
			Objects: []s3manager.BatchDeleteObject{
				{
					Object: &s3.DeleteObjectInput{
						Key:    aws.String(path),
						Bucket: aws.String(u.Bucket),
					},
				},
			},
		},
	)
}

func (u *S3Uploader) DeleteBulk(ctx context.Context, paths []string) error {
	if os.Getenv("DEBUG") != "" {
		return nil
	}

	objects := make([]s3manager.BatchDeleteObject, 0, len(paths))
	for _, path := range paths {
		objects = append(objects, s3manager.BatchDeleteObject{
			Object: &s3.DeleteObjectInput{
				Key:    aws.String(path),
				Bucket: aws.String(u.Bucket),
			},
		})
	}

	batcher := s3manager.NewBatchDelete(u.Session)
	return batcher.Delete(
		aws.BackgroundContext(),
		&s3manager.DeleteObjectsIterator{
			Objects: objects,
		},
	)
}
