package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// testCallToolParams is a helper type for test CallToolRequest params
type testCallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Meta      *struct {
		ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
	} `json:"_meta,omitempty"`
}

// testServer creates a test HTTP server that handles API requests
func testServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *TogglClient) {
	ts := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(ts.Close)

	customClient := &http.Client{
		Transport: &testTransport{testURL: ts.URL},
	}

	client := NewTogglClient("test-token", WithHTTPClient(customClient))
	return ts, client
}

// testTransport redirects requests to test server
type testTransport struct {
	testURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	testURL, err := url.Parse(t.testURL)
	if err != nil {
		return nil, err
	}

	req.URL.Scheme = testURL.Scheme
	req.URL.Host = testURL.Host
	return http.DefaultTransport.RoundTrip(req)
}

// Test fixtures
var (
	testUser = UserInfo{
		ID:                 123,
		Email:              "test@example.com",
		Fullname:           "Test User",
		DefaultWorkspaceID: 456,
	}

	testTimeEntry = TimeEntry{
		BaseEntity: BaseEntity{
			ID:          789,
			WorkspaceID: 456,
			At:          time.Now(),
		},
		Description: "Test Entry",
		Start:       time.Now().Add(-1 * time.Hour),
		Duration:    3600,
		ProjectID:   intPtr(111),
	}

	testProject = Project{
		BaseEntity: BaseEntity{
			ID:          111,
			WorkspaceID: 456,
			At:          time.Now(),
		},
		Name:   "Test Project",
		Active: true,
	}
)

// writeJSON writes JSON response
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(message))
}

func TestSetupTools(t *testing.T) {
	s := server.NewMCPServer(
		"test-server",
		"1.0.0",
		server.WithLogging(),
	)

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, nil)
	})

	err := SetupTools(s, client)
	if err != nil {
		t.Fatalf("SetupTools failed: %v", err)
	}

	// SetupTools should complete without error
	// Note: The server doesn't expose a way to verify registered tools
}

func TestHandleTestConnection(t *testing.T) {
	tests := []struct {
		name           string
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		expectedResult string
	}{
		{
			name: "successful connection",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v9/me" || r.Method != http.MethodGet {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				writeJSON(w, http.StatusOK, testUser)
			},
			expectedError: false,
			expectedResult: fmt.Sprintf(`✅ Authentication successful!
User: %s
Email: %s
Default Workspace ID: %d`, testUser.Fullname, testUser.Email, testUser.DefaultWorkspaceID),
		},
		{
			name: "authentication failure",
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeError(w, http.StatusUnauthorized, `{"error":"Invalid API token"}`)
			},
			expectedError:  false, // Returns error result, not actual error
			expectedResult: "Authentication failed:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)
			result, err := handleTestConnection(context.Background(), client, mcp.CallToolRequest{})

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.expectedError && result != nil {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, tt.expectedResult) {
					t.Errorf("expected result to contain %q, got %q", tt.expectedResult, content)
				}
			}
		})
	}
}

func TestHandleStartTimeEntry(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		validateResult func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "successful start",
			params: map[string]interface{}{
				"description":  "Test task",
				"workspace_id": float64(456),
				"project_id":   float64(111),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/workspaces/456/time_entries") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				var req TimeEntryRequest
				json.NewDecoder(r.Body).Decode(&req)
				if req.Description != "Test task" {
					t.Errorf("expected description 'Test task', got %s", req.Description)
				}

				writeJSON(w, http.StatusOK, testTimeEntry)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Started time entry: Test Entry") {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name: "missing description",
			params: map[string]interface{}{
				"workspace_id": float64(456),
			},
			expectedError: true,
		},
		{
			name: "missing workspace_id",
			params: map[string]interface{}{
				"description": "Test task",
			},
			expectedError: true,
		},
		{
			name: "API error",
			params: map[string]interface{}{
				"description":  "Test task",
				"workspace_id": float64(456),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeError(w, http.StatusBadRequest, `{"error":"Invalid project ID"}`)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				if !result.IsError {
					t.Error("expected error result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)

			req := mcp.CallToolRequest{
				Params: testCallToolParams{
					Arguments: tt.params,
				},
			}

			result, err := handleStartTimeEntry(context.Background(), client, req)

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil && result != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestHandleStopTimeEntry(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		validateResult func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "successful stop",
			params: map[string]interface{}{
				"workspace_id": float64(456),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, "/me/time_entries/current"):
					writeJSON(w, http.StatusOK, testTimeEntry)
				case strings.Contains(r.URL.Path, "/stop"):
					writeJSON(w, http.StatusOK, testTimeEntry)
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Stopped time entry") {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name: "no running entry",
			params: map[string]interface{}{
				"workspace_id": float64(456),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if content != "No running time entry found" {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name:          "missing workspace_id",
			params:        map[string]interface{}{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)

			req := mcp.CallToolRequest{
				Params: testCallToolParams{
					Arguments: tt.params,
				},
			}

			result, err := handleStopTimeEntry(context.Background(), client, req)

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil && result != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestHandleGetCurrentTimeEntry(t *testing.T) {
	tests := []struct {
		name           string
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		validateResult func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "running entry found",
			handler: func(w http.ResponseWriter, r *http.Request) {
				entry := testTimeEntry
				entry.Start = time.Now().Add(-30 * time.Minute)
				writeJSON(w, http.StatusOK, entry)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Current time entry: Test Entry") {
					t.Errorf("unexpected result: %s", content)
				}
				if !strings.Contains(content, "Running for:") {
					t.Error("expected duration in result")
				}
			},
		},
		{
			name: "no running entry",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if content != "No running time entry found" {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)

			result, err := handleGetCurrentTimeEntry(context.Background(), client, mcp.CallToolRequest{})

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil && result != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestHandleGetTimeEntries(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		validateResult func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name:   "get entries without date filter",
			params: map[string]interface{}{},
			handler: func(w http.ResponseWriter, r *http.Request) {
				entries := []TimeEntry{testTimeEntry, testTimeEntry}
				writeJSON(w, http.StatusOK, entries)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Found 2 time entries") {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name: "get entries with date range",
			params: map[string]interface{}{
				"start_date": "2025-01-15",
				"end_date":   "2025-01-16",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify query parameters
				if !strings.Contains(r.URL.RawQuery, "start_date=2025-01-15") {
					t.Error("expected start_date in query")
				}
				if !strings.Contains(r.URL.RawQuery, "end_date=2025-01-16") {
					t.Error("expected end_date in query")
				}
				writeJSON(w, http.StatusOK, []TimeEntry{testTimeEntry})
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Found 1 time entries") {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name: "no entries found with same date warning",
			params: map[string]interface{}{
				"start_date": "2025-01-15",
				"end_date":   "2025-01-15", // Same date
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, []TimeEntry{})
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Found 0 time entries") {
					t.Errorf("unexpected result: %s", content)
				}
				if !strings.Contains(content, "LIKELY ISSUE: You used the same date") {
					t.Error("expected warning about same date")
				}
			},
		},
		{
			name: "invalid date format",
			params: map[string]interface{}{
				"start_date": "15-01-2025", // Wrong format
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)

			req := mcp.CallToolRequest{
				Params: testCallToolParams{
					Arguments: tt.params,
				},
			}

			result, err := handleGetTimeEntries(context.Background(), client, req)

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil && result != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestHandleGetTimeEntriesForDay(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		validateResult func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "valid date",
			params: map[string]interface{}{
				"date": "2025-01-15",
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify it converts single date to range
				if !strings.Contains(r.URL.RawQuery, "start_date=2025-01-15") {
					t.Error("expected start_date=2025-01-15")
				}
				if !strings.Contains(r.URL.RawQuery, "end_date=2025-01-16") {
					t.Error("expected end_date=2025-01-16")
				}
				writeJSON(w, http.StatusOK, []TimeEntry{testTimeEntry})
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Found 1 time entries") {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name:          "missing date",
			params:        map[string]interface{}{},
			expectedError: true,
		},
		{
			name: "invalid date format",
			params: map[string]interface{}{
				"date": "15-01-2025",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)

			req := mcp.CallToolRequest{
				Params: testCallToolParams{
					Arguments: tt.params,
				},
			}

			result, err := handleGetTimeEntriesForDay(context.Background(), client, req)

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil && result != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestHandleCreateProject(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		validateResult func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "successful project creation",
			params: map[string]interface{}{
				"name":         "New Project",
				"workspace_id": float64(456),
				"color":        "#FF0000",
				"client_id":    float64(789),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/workspaces/456/projects") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				var req map[string]interface{}
				json.NewDecoder(r.Body).Decode(&req)
				if req["name"] != "New Project" {
					t.Errorf("expected name 'New Project', got %v", req["name"])
				}

				writeJSON(w, http.StatusOK, testProject)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Created project: Test Project") {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name: "missing required name",
			params: map[string]interface{}{
				"workspace_id": float64(456),
			},
			expectedError: true,
		},
		{
			name: "missing required workspace_id",
			params: map[string]interface{}{
				"name": "New Project",
			},
			expectedError: true,
		},
		{
			name: "API error",
			params: map[string]interface{}{
				"name":         "New Project",
				"workspace_id": float64(456),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeError(w, http.StatusBadRequest, `{"error":"Project name already exists"}`)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				if !result.IsError {
					t.Error("expected error result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)

			req := mcp.CallToolRequest{
				Params: testCallToolParams{
					Arguments: tt.params,
				},
			}

			result, err := handleCreateProject(context.Background(), client, req)

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil && result != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestHandleGetProjects(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]interface{}
		handler        func(w http.ResponseWriter, r *http.Request)
		expectedError  bool
		validateResult func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "get all projects",
			params: map[string]interface{}{
				"workspace_id": float64(456),
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				projects := []Project{
					{
						BaseEntity: BaseEntity{ID: 1, WorkspaceID: 456},
						Name:       "Project 1",
						Active:     true,
					},
					{
						BaseEntity: BaseEntity{ID: 2, WorkspaceID: 456},
						Name:       "Project 2",
						Active:     false,
					},
				}
				writeJSON(w, http.StatusOK, projects)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Found 2 projects") {
					t.Errorf("unexpected result: %s", content)
				}
				if !strings.Contains(content, "Project 1 (ID: 1, active)") {
					t.Error("expected active project in result")
				}
				if !strings.Contains(content, "Project 2 (ID: 2, inactive)") {
					t.Error("expected inactive project in result")
				}
			},
		},
		{
			name: "filter active projects",
			params: map[string]interface{}{
				"workspace_id": float64(456),
				"active":       true,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Verify active parameter
				if !strings.Contains(r.URL.RawQuery, "active=true") {
					t.Error("expected active=true in query")
				}
				projects := []Project{
					{
						BaseEntity: BaseEntity{ID: 1, WorkspaceID: 456},
						Name:       "Project 1",
						Active:     true,
					},
				}
				writeJSON(w, http.StatusOK, projects)
			},
			expectedError: false,
			validateResult: func(t *testing.T, result *mcp.CallToolResult) {
				content := result.Content[0].(mcp.TextContent).Text
				if !strings.Contains(content, "Found 1 projects") {
					t.Errorf("unexpected result: %s", content)
				}
			},
		},
		{
			name:          "missing workspace_id",
			params:        map[string]interface{}{},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := testServer(t, tt.handler)

			req := mcp.CallToolRequest{
				Params: testCallToolParams{
					Arguments: tt.params,
				},
			}

			result, err := handleGetProjects(context.Background(), client, req)

			if tt.expectedError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validateResult != nil && result != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

// Test error handling
func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 401,
		Body:       `{"error":"Unauthorized"}`,
	}

	expectedMsg := `API error (status 401): {"error":"Unauthorized"}`
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}

	// Test Is method
	if !err.Is(ErrAPIRequest) {
		t.Error("APIError should match ErrAPIRequest")
	}
}
