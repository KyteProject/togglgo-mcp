package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	togglAPIBase   = "https://api.track.toggl.com/api/v9"
	defaultTimeout = 30 * time.Second
)

// ClientOption is a functional option for configuring the client
type ClientOption func(*TogglClient)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *TogglClient) {
		c.client = client
	}
}

// WithLogger sets a custom logger
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *TogglClient) {
		c.logger = logger
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *TogglClient) {
		c.client.Timeout = timeout
	}
}

// TogglClient represents a client for the Toggl API
type TogglClient struct {
	APIToken string
	client   *http.Client
	logger   *slog.Logger
}

// NewTogglClient creates a new Toggl client with options
func NewTogglClient(apiToken string, opts ...ClientOption) *TogglClient {
	c := &TogglClient{
		APIToken: apiToken,
		client:   &http.Client{Timeout: defaultTimeout},
		logger:   slog.Default(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// makeRequest is a generic method for making API requests
func (c *TogglClient) makeRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, togglAPIBase+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.APIToken, "api_token")

	c.logger.Debug("making API request",
		slog.String("method", method),
		slog.String("endpoint", endpoint),
	)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

// decodeResponse is a generic function to decode JSON responses
func decodeResponse[T any](resp *http.Response) (T, error) {
	var result T
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return result, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return result, fmt.Errorf("decoding response: %w", err)
	}

	return result, nil
}
