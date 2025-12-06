package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	client     *minio.Client
	bucketName string
}

// NewS3Storage initializes a MinIO/S3 client and ensures the bucket exists.
func NewS3Storage(endpoint, region, accessKey, secretKey, bucket string, useSSL bool) (*S3Storage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
		log.Printf("created bucket %s", bucket)
	}

	return &S3Storage{client: client, bucketName: bucket}, nil
}

// Upload streams content to the configured bucket and returns the object key.
func (s *S3Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("missing key")
	}

	_, err := s.client.PutObject(ctx, s.bucketName, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	return key, nil
}
