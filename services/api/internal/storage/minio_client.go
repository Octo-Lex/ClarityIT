package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps minio-go to implement the S3Client interface.
type MinIOClient struct {
	client *minio.Client
	logger *slog.Logger
}

// NewMinIOClient creates a real MinIO/S3 client.
func NewMinIOClient(endpoint string, accessKey, secretKey string, useSSL bool) (*MinIOClient, error) {
	// Strip any scheme from endpoint — minio-go wants bare host:port
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIOClient{
		client: client,
		logger: slog.Default(),
	}, nil
}

func (m *MinIOClient) PutObject(ctx context.Context, bucket, key string, data []byte, contentType string) error {
	reader := io.NopCloser(strings.NewReader(string(data)))
	defer reader.Close()

	_, err := m.client.PutObject(ctx, bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio put object: %w", err)
	}
	return nil
}

func (m *MinIOClient) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := m.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("minio presigned url: %w", err)
	}
	return u.String(), nil
}

// EnsureBucket creates the bucket if it doesn't exist.
func (m *MinIOClient) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := m.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		err = m.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		m.logger.Info("created MinIO bucket", "bucket", bucket)
	}
	return nil
}

// Endpoint returns the configured endpoint URL for health checks.
func (m *MinIOClient) Endpoint() string {
	return m.client.EndpointURL().String()
}

// Helper to parse endpoint into URL (for presigning reference).
func parseEndpoint(raw string) (*url.URL, error) {
	if !strings.HasPrefix(raw, "http") {
		raw = "http://" + raw
	}
	return url.Parse(raw)
}
