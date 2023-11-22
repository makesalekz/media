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
)

type S3Uploader struct {
	Bucket  string
	Session *session.Session
}

// NewS3Uploader .
func NewS3Uploader(c *Config) (*S3Uploader, error) {
	uploader := &S3Uploader{}

	if os.Getenv("DEBUG") == "" {
		region, err := c.Value("AWS_REGION").String()
		if err != nil {
			return nil, fmt.Errorf("failed to get AWS_REGION: %v", err)
		}
		bucket, err := c.Value("AWS_BUCKET").String()
		if err != nil {
			return nil, fmt.Errorf("failed to get AWS_BUCKET: %v", err)
		}

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(region),
		})
		if err != nil {
			return nil, fmt.Errorf("AWS Session error: %v", err)
		}

		uploader = &S3Uploader{
			Bucket:  bucket,
			Session: sess,
		}
	}

	return uploader, nil
}

func (u *S3Uploader) Upload(ctx context.Context, path string, fileData []byte, mimeType string) (string, error) {
	uploader := s3manager.NewUploader(u.Session)
	out, err := uploader.Upload(&s3manager.UploadInput{
		ACL:         aws.String("public-read"),
		Bucket:      aws.String(u.Bucket),
		Key:         aws.String(path),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return "", err
	}

	return out.Location, nil
}

func (u *S3Uploader) Delete(ctx context.Context, path string) error {
	batcher := s3manager.NewBatchDelete(u.Session)
	return batcher.Delete(aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
		Objects: []s3manager.BatchDeleteObject{
			{
				Object: &s3.DeleteObjectInput{
					Key:    aws.String(path),
					Bucket: aws.String(u.Bucket),
				},
			},
		},
	})
}
