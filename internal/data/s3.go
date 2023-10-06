package data

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"google.golang.org/genproto/googleapis/api/httpbody"
)

type S3Uploader struct {
	bucket  string
	session *session.Session
}

// NewS3Uploader .
func NewS3Uploader(c *Config) (*S3Uploader, error) {
	region, err := c.Value("AWS_REGION").String()
	if err != nil {
		return nil, err
	}
	bucket, err := c.Value("AWS_BUCKET").String()
	if err != nil {
		return nil, err
	}
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, fmt.Errorf("AWS Session error: %v", err)
	}

	return &S3Uploader{
		bucket:  bucket,
		session: sess,
	}, nil
}

func (u *S3Uploader) Upload(ctx context.Context, path string, file *httpbody.HttpBody) (string, error) {
	uploader := s3manager.NewUploader(u.session)
	out, err := uploader.Upload(&s3manager.UploadInput{
		ACL:         aws.String("public-read"),
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(path),
		Body:        bytes.NewReader(file.GetData()),
		ContentType: aws.String(file.GetContentType()),
	})
	if err != nil {
		return "", err
	}

	return out.Location, nil
}

func (u *S3Uploader) Delete(ctx context.Context, path string) error {
	batcher := s3manager.NewBatchDelete(u.session)
	return batcher.Delete(aws.BackgroundContext(), &s3manager.DeleteObjectsIterator{
		Objects: []s3manager.BatchDeleteObject{
			{
				Object: &s3.DeleteObjectInput{
					Key:    aws.String(path),
					Bucket: aws.String(u.bucket),
				},
			},
		},
	})
}
