package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3Client(endpoint, region, bucket, accessKey, secretKey string) *S3Client {
	if region == "" {
		region = "us-east-1"
	}
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		EndpointResolverWithOptions: aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: endpoint}, nil
			},
		),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	return &S3Client{client: client, bucket: bucket}
}

// PresignUpload returns a presigned PUT URL for the given key, valid for 15 minutes.
func (s *S3Client) PresignUpload(ctx context.Context, key, contentType string) (string, error) {
	presignClient := s3.NewPresignClient(s.client)
	req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(15*time.Minute))
	if err != nil {
		return "", fmt.Errorf("presign upload: %w", err)
	}
	return req.URL, nil
}

// PublicURL returns the public URL for an object.
func (s *S3Client) PublicURL(endpoint, key string) string {
	return fmt.Sprintf("%s/%s/%s", endpoint, s.bucket, key)
}
