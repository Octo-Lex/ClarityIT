package presenton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the interface for talking to the Presenton service.
type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
	DownloadFile(ctx context.Context, presentationID, format string) ([]byte, string, error)
}

// GenerateRequest is the payload sent to Presenton.
type GenerateRequest struct {
	Content     string `json:"content"`
	NumSlides   int    `json:"n_slides"`
	Template    string `json:"template"`
	Tone        string `json:"tone"`
	Language    string `json:"language"`
	ExportAs    string `json:"export_as"`
	Instructions string `json:"instructions,omitempty"`
}

// GenerateResponse is what Presenton returns after generation.
type GenerateResponse struct {
	PresentationID string `json:"presentation_id"`
	Path           string `json:"path"`
	EditPath       string `json:"edit_path"`
}

// httpClient implements Client using HTTP Basic auth.
type httpClient struct {
	baseURL  string
	username string
	password string
	timeout  time.Duration
	client   *http.Client
}

// NewClient creates a Presenton HTTP client.
func NewClient(baseURL, username, password string, timeout time.Duration) Client {
	return &httpClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		timeout:  timeout,
		client:   &http.Client{Timeout: timeout},
	}
}

func (c *httpClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/api/v1/ppt/presentation/generate"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.SetBasicAuth(c.username, c.password)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("presenton request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("presenton returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func (c *httpClient) DownloadFile(ctx context.Context, presentationID, format string) ([]byte, string, error) {
	// Use the file download endpoint
	url := fmt.Sprintf("%s/api/v1/ppt/presentation/%s/file?format=%s", c.baseURL, presentationID, format)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create download request: %w", err)
	}
	httpReq.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, "", fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 200*1024*1024)) // 200MB hard cap
	if err != nil {
		return nil, "", fmt.Errorf("read download body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return data, contentType, nil
}
