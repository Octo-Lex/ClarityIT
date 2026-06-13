package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// MinIOHealthChecker checks MinIO connectivity by making an HTTP request
// to the MinIO endpoint. Since the main app doesn't have aws-sdk-go-v2
// as a dependency, we use a simple HTTP health check.
type MinIOHealthChecker struct {
	endpoint string
	secure   bool
	client   *http.Client
}

func NewMinIOHealthChecker(endpoint string, useSSL bool) *MinIOHealthChecker {
	return &MinIOHealthChecker{
		endpoint: endpoint,
		secure:   useSSL,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// HeadBucket checks if MinIO is responsive by requesting the health endpoint.
func (m *MinIOHealthChecker) HeadBucket(ctx context.Context, bucket string) error {
	scheme := "http"
	if m.secure {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/minio/health/live", scheme, m.endpoint)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("MinIO health check returned %d", resp.StatusCode)
	}
	return nil
}
