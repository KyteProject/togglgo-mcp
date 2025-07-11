package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewTogglClient(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		client := NewTogglClient("test-token")

		if client.APIToken != "test-token" {
			t.Errorf("expected APIToken 'test-token', got %s", client.APIToken)
		}
		if client.client == nil {
			t.Error("expected HTTP client to be set")
		}
		if client.client.Timeout != defaultTimeout {
			t.Errorf("expected timeout %v, got %v", defaultTimeout, client.client.Timeout)
		}
		if client.logger == nil {
			t.Error("expected logger to be set")
		}
	})

	t.Run("with custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{Timeout: 5 * time.Second}
		client := NewTogglClient("test-token", WithHTTPClient(customClient))

		if client.client != customClient {
			t.Error("expected custom HTTP client to be set")
		}
		if client.client.Timeout != 5*time.Second {
			t.Errorf("expected timeout 5s, got %v", client.client.Timeout)
		}
	})

	t.Run("with custom logger", func(t *testing.T) {
		customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		client := NewTogglClient("test-token", WithLogger(customLogger))

		if client.logger != customLogger {
			t.Error("expected custom logger to be set")
		}
	})

	t.Run("with custom timeout", func(t *testing.T) {
		customTimeout := 10 * time.Second
		client := NewTogglClient("test-token", WithTimeout(customTimeout))

		if client.client.Timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, client.client.Timeout)
		}
	})

	t.Run("with multiple options", func(t *testing.T) {
		customClient := &http.Client{}
		customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		customTimeout := 15 * time.Second

		client := NewTogglClient("test-token",
			WithHTTPClient(customClient),
			WithLogger(customLogger),
			WithTimeout(customTimeout),
		)

		if client.client != customClient {
			t.Error("expected custom HTTP client to be set")
		}
		if client.logger != customLogger {
			t.Error("expected custom logger to be set")
		}
		if client.client.Timeout != customTimeout {
			t.Errorf("expected timeout %v, got %v", customTimeout, client.client.Timeout)
		}
	})
}

func TestClientOptions(t *testing.T) {
	t.Run("WithHTTPClient", func(t *testing.T) {
		customClient := &http.Client{Timeout: 1 * time.Second}
		client := &TogglClient{}

		opt := WithHTTPClient(customClient)
		opt(client)

		if client.client != customClient {
			t.Error("WithHTTPClient option didn't set the client")
		}
	})

	t.Run("WithLogger", func(t *testing.T) {
		customLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
		client := &TogglClient{}

		opt := WithLogger(customLogger)
		opt(client)

		if client.logger != customLogger {
			t.Error("WithLogger option didn't set the logger")
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		timeout := 5 * time.Second
		client := &TogglClient{client: &http.Client{}}

		opt := WithTimeout(timeout)
		opt(client)

		if client.client.Timeout != timeout {
			t.Errorf("WithTimeout option didn't set timeout, expected %v, got %v",
				timeout, client.client.Timeout)
		}
	})
}

func TestTogglClient_makeRequest(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		endpoint        string
		body            io.Reader
		handler         func(w http.ResponseWriter, r *http.Request)
		expectedError   bool
		validateRequest func(t *testing.T, r *http.Request)
	}{
		{
			name:     "successful GET request",
			method:   "GET",
			endpoint: "/me",
			body:     nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 123}`))
			},
			expectedError: false,
			validateRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/api/v9/me" {
					t.Errorf("expected /api/v9/me, got %s", r.URL.Path)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("expected Content-Type header to be application/json")
				}

				// Check basic auth
				username, password, ok := r.BasicAuth()
				if !ok {
					t.Error("expected basic auth to be set")
				}
				if username != "test-token" || password != "api_token" {
					t.Errorf("expected basic auth test-token:api_token, got %s:%s",
						username, password)
				}
			},
		},
		{
			name:     "successful POST request with body",
			method:   "POST",
			endpoint: "/time_entries",
			body:     strings.NewReader(`{"description": "test"}`),
			handler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if string(body) != `{"description": "test"}` {
					t.Errorf("unexpected body: %s", string(body))
				}
				w.WriteHeader(http.StatusCreated)
			},
			expectedError: false,
			validateRequest: func(t *testing.T, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
			},
		},
		{
			name:     "request with context cancellation",
			method:   "GET",
			endpoint: "/slow",
			body:     nil,
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Simulate slow response
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.validateRequest != nil {
					tt.validateRequest(t, r)
				}
				tt.handler(w, r)
			}))
			defer ts.Close()

			// Create client with custom transport to redirect to test server
			client := NewTogglClient("test-token", WithHTTPClient(&http.Client{
				Transport: &testTransport{testURL: ts.URL},
			}))

			// Create context
			ctx := context.Background()
			if tt.name == "request with context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			// Make request
			resp, err := client.makeRequest(ctx, tt.method, tt.endpoint, tt.body)

			// Check results
			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.expectedError && resp == nil {
				t.Fatal("expected response but got nil")
			}
			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

func TestTogglClient_makeRequest_InvalidEndpoint(t *testing.T) {
	client := NewTogglClient("test-token")

	// Test with invalid URL characters
	_, err := client.makeRequest(context.Background(), "GET", "\x00", nil)
	if err == nil {
		t.Error("expected error for invalid endpoint")
	}
	if !strings.Contains(err.Error(), "creating request") {
		t.Errorf("expected 'creating request' error, got: %v", err)
	}
}

func TestDecodeResponse(t *testing.T) {
	type testResponse struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	tests := []struct {
		name           string
		statusCode     int
		body           string
		expectedResult testResponse
		expectedError  bool
		errorType      string
	}{
		{
			name:       "successful response",
			statusCode: 200,
			body:       `{"id": 123, "name": "test"}`,
			expectedResult: testResponse{
				ID:   123,
				Name: "test",
			},
			expectedError: false,
		},
		{
			name:           "empty successful response",
			statusCode:     200,
			body:           `{}`,
			expectedResult: testResponse{},
			expectedError:  false,
		},
		{
			name:          "API error response",
			statusCode:    400,
			body:          `{"error": "Bad Request"}`,
			expectedError: true,
			errorType:     "APIError",
		},
		{
			name:          "server error response",
			statusCode:    500,
			body:          `{"error": "Internal Server Error"}`,
			expectedError: true,
			errorType:     "APIError",
		},
		{
			name:          "unauthorized response",
			statusCode:    401,
			body:          `{"error": "Unauthorized"}`,
			expectedError: true,
			errorType:     "APIError",
		},
		{
			name:          "not found response",
			statusCode:    404,
			body:          `{"error": "Not Found"}`,
			expectedError: true,
			errorType:     "APIError",
		},
		{
			name:          "invalid JSON response",
			statusCode:    200,
			body:          `{"invalid": json}`,
			expectedError: true,
			errorType:     "decode",
		},
		{
			name:          "malformed JSON response",
			statusCode:    200,
			body:          `{broken json`,
			expectedError: true,
			errorType:     "decode",
		},
		{
			name:          "unexpected JSON structure",
			statusCode:    200,
			body:          `"not an object"`,
			expectedError: true,
			errorType:     "decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock response
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
				Header:     make(http.Header),
			}

			// Test decode
			result, err := decodeResponse[testResponse](resp)

			// Check error expectation
			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check error type
			if tt.expectedError && tt.errorType != "" {
				switch tt.errorType {
				case "APIError":
					var apiErr *APIError
					if !errors.As(err, &apiErr) {
						t.Errorf("expected APIError, got %T: %v", err, err)
					} else {
						if apiErr.StatusCode != tt.statusCode {
							t.Errorf("expected status code %d, got %d",
								tt.statusCode, apiErr.StatusCode)
						}
						if apiErr.Body != tt.body {
							t.Errorf("expected body %q, got %q", tt.body, apiErr.Body)
						}
					}
				case "decode":
					if !strings.Contains(err.Error(), "decoding response") {
						t.Errorf("expected decode error, got: %v", err)
					}
				}
			}

			// Check successful result
			if !tt.expectedError {
				if result.ID != tt.expectedResult.ID {
					t.Errorf("expected ID %d, got %d", tt.expectedResult.ID, result.ID)
				}
				if result.Name != tt.expectedResult.Name {
					t.Errorf("expected Name %s, got %s", tt.expectedResult.Name, result.Name)
				}
			}
		})
	}
}

func TestDecodeResponse_ReadBodyError(t *testing.T) {
	// Create response with error reader
	resp := &http.Response{
		StatusCode: 200,
		Body:       &errorReader{},
		Header:     make(http.Header),
	}

	_, err := decodeResponse[map[string]interface{}](resp)
	if err == nil {
		t.Fatal("expected error from reading body")
	}
	if !strings.Contains(err.Error(), "reading response body") {
		t.Errorf("expected 'reading response body' error, got: %v", err)
	}
}

func TestDecodeResponse_DifferentTypes(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		testType string
	}{
		{
			name:     "string type",
			body:     `"test string"`,
			testType: "string",
		},
		{
			name:     "integer type",
			body:     `42`,
			testType: "int",
		},
		{
			name:     "boolean type",
			body:     `true`,
			testType: "bool",
		},
		{
			name:     "array type",
			body:     `[1, 2, 3]`,
			testType: "array",
		},
		{
			name:     "map type",
			body:     `{"key": "value"}`,
			testType: "map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
				Header:     make(http.Header),
			}

			switch tt.testType {
			case "string":
				result, err := decodeResponse[string](resp)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != "test string" {
					t.Errorf("expected 'test string', got %s", result)
				}
			case "int":
				result, err := decodeResponse[int](resp)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != 42 {
					t.Errorf("expected 42, got %d", result)
				}
			case "bool":
				result, err := decodeResponse[bool](resp)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != true {
					t.Errorf("expected true, got %t", result)
				}
			case "array":
				result, err := decodeResponse[[]int](resp)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				expected := []int{1, 2, 3}
				if len(result) != len(expected) {
					t.Errorf("expected %v, got %v", expected, result)
				}
				for i, v := range expected {
					if result[i] != v {
						t.Errorf("expected %v, got %v", expected, result)
						break
					}
				}
			case "map":
				result, err := decodeResponse[map[string]string](resp)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result["key"] != "value" {
					t.Errorf("expected map[key:value], got %v", result)
				}
			}
		})
	}
}

// errorReader implements io.ReadCloser and always returns an error
type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) {
	return 0, fmt.Errorf("read error")
}

func (e *errorReader) Close() error {
	return nil
}

func TestConstants(t *testing.T) {
	t.Run("togglAPIBase", func(t *testing.T) {
		expected := "https://api.track.toggl.com/api/v9"
		if togglAPIBase != expected {
			t.Errorf("expected togglAPIBase %s, got %s", expected, togglAPIBase)
		}
	})

	t.Run("defaultTimeout", func(t *testing.T) {
		expected := 30 * time.Second
		if defaultTimeout != expected {
			t.Errorf("expected defaultTimeout %v, got %v", expected, defaultTimeout)
		}
	})
}

func TestTogglClient_Integration(t *testing.T) {
	// Test full integration with a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type: application/json")
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != "integration-token" || password != "api_token" {
			t.Error("expected proper basic auth")
		}

		// Return mock response
		response := map[string]interface{}{
			"id":   123,
			"name": "Integration Test",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	// Create client
	client := NewTogglClient("integration-token", WithHTTPClient(&http.Client{
		Transport: &testTransport{testURL: ts.URL},
	}))

	// Make request
	resp, err := client.makeRequest(context.Background(), "GET", "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	// Decode response
	type integrationResponse struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	result, err := decodeResponse[integrationResponse](resp)
	if err != nil {
		t.Fatalf("unexpected error decoding: %v", err)
	}

	if result.ID != 123 {
		t.Errorf("expected ID 123, got %d", result.ID)
	}
	if result.Name != "Integration Test" {
		t.Errorf("expected name 'Integration Test', got %s", result.Name)
	}
}
