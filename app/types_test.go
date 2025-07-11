package app

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// Test custom error variables
func TestCustomErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedMsg string
	}{
		{
			name:        "ErrNoAPIToken",
			err:         ErrNoAPIToken,
			expectedMsg: "TOGGL_API_TOKEN environment variable is required",
		},
		{
			name:        "ErrInvalidWorkspace",
			err:         ErrInvalidWorkspace,
			expectedMsg: "workspace_id must be a number",
		},
		{
			name:        "ErrInvalidProject",
			err:         ErrInvalidProject,
			expectedMsg: "project_id must be a number",
		},
		{
			name:        "ErrInvalidDate",
			err:         ErrInvalidDate,
			expectedMsg: "invalid date format",
		},
		{
			name:        "ErrNoRunningEntry",
			err:         ErrNoRunningEntry,
			expectedMsg: "no running time entry found",
		},
		{
			name:        "ErrAPIRequest",
			err:         ErrAPIRequest,
			expectedMsg: "API request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMsg {
				t.Errorf("%s.Error() = %v, want %v", tt.name, tt.err.Error(), tt.expectedMsg)
			}
		})
	}
}

// Test APIError struct
func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{
			name:       "basic error",
			statusCode: 404,
			body:       "Not Found",
			want:       "API error (status 404): Not Found",
		},
		{
			name:       "server error",
			statusCode: 500,
			body:       "Internal Server Error",
			want:       "API error (status 500): Internal Server Error",
		},
		{
			name:       "empty body",
			statusCode: 400,
			body:       "",
			want:       "API error (status 400): ",
		},
		{
			name:       "zero status code",
			statusCode: 0,
			body:       "Connection failed",
			want:       "API error (status 0): Connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &APIError{
				StatusCode: tt.statusCode,
				Body:       tt.body,
			}
			if got := e.Error(); got != tt.want {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAPIError_Is(t *testing.T) {
	tests := []struct {
		name   string
		target error
		want   bool
	}{
		{
			name:   "matches ErrAPIRequest",
			target: ErrAPIRequest,
			want:   true,
		},
		{
			name:   "does not match ErrNoAPIToken",
			target: ErrNoAPIToken,
			want:   false,
		},
		{
			name:   "does not match custom error",
			target: errors.New("custom error"),
			want:   false,
		},
		{
			name:   "does not match nil",
			target: nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &APIError{StatusCode: 500, Body: "test"}
			if got := e.Is(tt.target); got != tt.want {
				t.Errorf("APIError.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test BaseEntity struct
func TestBaseEntity(t *testing.T) {
	now := time.Now()
	entity := BaseEntity{
		ID:          123,
		WorkspaceID: 456,
		At:          now,
	}

	if entity.ID != 123 {
		t.Errorf("BaseEntity.ID = %v, want %v", entity.ID, 123)
	}
	if entity.WorkspaceID != 456 {
		t.Errorf("BaseEntity.WorkspaceID = %v, want %v", entity.WorkspaceID, 456)
	}
	if !entity.At.Equal(now) {
		t.Errorf("BaseEntity.At = %v, want %v", entity.At, now)
	}
}

// Test TimeEntry struct
func TestTimeEntry(t *testing.T) {
	now := time.Now()
	stopTime := now.Add(time.Hour)
	projectID := 789

	entry := TimeEntry{
		BaseEntity: BaseEntity{
			ID:          1,
			WorkspaceID: 2,
			At:          now,
		},
		ProjectID:   &projectID,
		Description: "Test task",
		Start:       now,
		Stop:        &stopTime,
		Duration:    3600,
		Tags:        []string{"tag1", "tag2"},
		TagIDs:      []int{10, 20},
	}

	if entry.ID != 1 {
		t.Errorf("TimeEntry.ID = %v, want %v", entry.ID, 1)
	}
	if entry.ProjectID == nil || *entry.ProjectID != 789 {
		t.Errorf("TimeEntry.ProjectID = %v, want %v", entry.ProjectID, &projectID)
	}
	if entry.Description != "Test task" {
		t.Errorf("TimeEntry.Description = %v, want %v", entry.Description, "Test task")
	}
	if entry.Duration != 3600 {
		t.Errorf("TimeEntry.Duration = %v, want %v", entry.Duration, 3600)
	}
	if len(entry.Tags) != 2 || entry.Tags[0] != "tag1" || entry.Tags[1] != "tag2" {
		t.Errorf("TimeEntry.Tags = %v, want %v", entry.Tags, []string{"tag1", "tag2"})
	}
	if len(entry.TagIDs) != 2 || entry.TagIDs[0] != 10 || entry.TagIDs[1] != 20 {
		t.Errorf("TimeEntry.TagIDs = %v, want %v", entry.TagIDs, []int{10, 20})
	}
	if !entry.Start.Equal(now) {
		t.Errorf("TimeEntry.Start = %v, want %v", entry.Start, now)
	}
	if entry.Stop == nil || !entry.Stop.Equal(stopTime) {
		t.Errorf("TimeEntry.Stop = %v, want %v", entry.Stop, &stopTime)
	}
}

func TestTimeEntry_NilProjectID(t *testing.T) {
	now := time.Now()
	entry := TimeEntry{
		BaseEntity: BaseEntity{
			ID:          1,
			WorkspaceID: 2,
			At:          now,
		},
		ProjectID:   nil,
		Description: "Test without project",
		Start:       now,
		Stop:        nil,
		Duration:    -1, // Running entry
	}

	if entry.ProjectID != nil {
		t.Errorf("TimeEntry.ProjectID = %v, want nil", entry.ProjectID)
	}
	if entry.Stop != nil {
		t.Errorf("TimeEntry.Stop = %v, want nil", entry.Stop)
	}
	if entry.Duration != -1 {
		t.Errorf("TimeEntry.Duration = %v, want %v", entry.Duration, -1)
	}
	if !entry.Start.Equal(now) {
		t.Errorf("TimeEntry.Start = %v, want %v", entry.Start, now)
	}
	if entry.Description != "Test without project" {
		t.Errorf("TimeEntry.Description = %v, want %v", entry.Description, "Test without project")
	}
	if entry.ID != 1 {
		t.Errorf("TimeEntry.ID = %v, want %v", entry.ID, 1)
	}
	if entry.WorkspaceID != 2 {
		t.Errorf("TimeEntry.WorkspaceID = %v, want %v", entry.WorkspaceID, 2)
	}
	if !entry.At.Equal(now) {
		t.Errorf("TimeEntry.At = %v, want %v", entry.At, now)
	}
}

// Test Project struct
func TestProject(t *testing.T) {
	now := time.Now()
	clientID := 999

	project := Project{
		BaseEntity: BaseEntity{
			ID:          100,
			WorkspaceID: 200,
			At:          now,
		},
		Name:     "Test Project",
		Active:   true,
		Color:    "#FF0000",
		ClientID: &clientID,
	}

	if project.ID != 100 {
		t.Errorf("Project.ID = %v, want %v", project.ID, 100)
	}
	if project.Name != "Test Project" {
		t.Errorf("Project.Name = %v, want %v", project.Name, "Test Project")
	}
	if !project.Active {
		t.Errorf("Project.Active = %v, want %v", project.Active, true)
	}
	if project.Color != "#FF0000" {
		t.Errorf("Project.Color = %v, want %v", project.Color, "#FF0000")
	}
	if project.ClientID == nil || *project.ClientID != 999 {
		t.Errorf("Project.ClientID = %v, want %v", project.ClientID, &clientID)
	}
}

func TestProject_NilClientID(t *testing.T) {
	now := time.Now()
	project := Project{
		BaseEntity: BaseEntity{
			ID:          100,
			WorkspaceID: 200,
			At:          now,
		},
		Name:     "Test Project",
		Active:   false,
		Color:    "",
		ClientID: nil,
	}

	if project.ClientID != nil {
		t.Errorf("Project.ClientID = %v, want nil", project.ClientID)
	}
	if project.Active {
		t.Errorf("Project.Active = %v, want %v", project.Active, false)
	}
	if project.Color != "" {
		t.Errorf("Project.Color = %v, want empty string", project.Color)
	}
	if project.ID != 100 {
		t.Errorf("Project.ID = %v, want %v", project.ID, 100)
	}
	if project.WorkspaceID != 200 {
		t.Errorf("Project.WorkspaceID = %v, want %v", project.WorkspaceID, 200)
	}
	if !project.At.Equal(now) {
		t.Errorf("Project.At = %v, want %v", project.At, now)
	}
	if project.Name != "Test Project" {
		t.Errorf("Project.Name = %v, want %v", project.Name, "Test Project")
	}
}

// Test TimeEntryRequest struct
func TestTimeEntryRequest(t *testing.T) {
	now := time.Now()
	projectID := 123

	request := TimeEntryRequest{
		Description: "Test request",
		Start:       now,
		Duration:    1800,
		ProjectID:   &projectID,
		Tags:        []string{"work", "important"},
		CreatedWith: "togglgo-mcp",
	}

	if request.Description != "Test request" {
		t.Errorf("TimeEntryRequest.Description = %v, want %v", request.Description, "Test request")
	}
	if !request.Start.Equal(now) {
		t.Errorf("TimeEntryRequest.Start = %v, want %v", request.Start, now)
	}
	if request.Duration != 1800 {
		t.Errorf("TimeEntryRequest.Duration = %v, want %v", request.Duration, 1800)
	}
	if request.ProjectID == nil || *request.ProjectID != 123 {
		t.Errorf("TimeEntryRequest.ProjectID = %v, want %v", request.ProjectID, &projectID)
	}
	if len(request.Tags) != 2 || request.Tags[0] != "work" || request.Tags[1] != "important" {
		t.Errorf("TimeEntryRequest.Tags = %v, want %v", request.Tags, []string{"work", "important"})
	}
	if request.CreatedWith != "togglgo-mcp" {
		t.Errorf("TimeEntryRequest.CreatedWith = %v, want %v", request.CreatedWith, "togglgo-mcp")
	}
}

func TestTimeEntryRequest_NilProjectID(t *testing.T) {
	now := time.Now()
	request := TimeEntryRequest{
		Description: "Test without project",
		Start:       now,
		Duration:    900,
		ProjectID:   nil,
		Tags:        nil,
		CreatedWith: "test",
	}

	if request.ProjectID != nil {
		t.Errorf("TimeEntryRequest.ProjectID = %v, want nil", request.ProjectID)
	}
	if request.Tags != nil {
		t.Errorf("TimeEntryRequest.Tags = %v, want nil", request.Tags)
	}
	if !request.Start.Equal(now) {
		t.Errorf("TimeEntryRequest.Start = %v, want %v", request.Start, now)
	}
	if request.CreatedWith != "test" {
		t.Errorf("TimeEntryRequest.CreatedWith = %v, want %v", request.CreatedWith, "test")
	}
	if request.Description != "Test without project" {
		t.Errorf("TimeEntryRequest.Description = %v, want %v", request.Description, "Test without project")
	}
	if request.Duration != 900 {
		t.Errorf("TimeEntryRequest.Duration = %v, want %v", request.Duration, 900)
	}
}

// Test UserInfo struct
func TestUserInfo(t *testing.T) {
	user := UserInfo{
		ID:                 42,
		Email:              "test@example.com",
		Fullname:           "Test User",
		DefaultWorkspaceID: 100,
	}

	if user.ID != 42 {
		t.Errorf("UserInfo.ID = %v, want %v", user.ID, 42)
	}
	if user.Email != "test@example.com" {
		t.Errorf("UserInfo.Email = %v, want %v", user.Email, "test@example.com")
	}
	if user.Fullname != "Test User" {
		t.Errorf("UserInfo.Fullname = %v, want %v", user.Fullname, "Test User")
	}
	if user.DefaultWorkspaceID != 100 {
		t.Errorf("UserInfo.DefaultWorkspaceID = %v, want %v", user.DefaultWorkspaceID, 100)
	}
}

// Test JSON marshaling/unmarshaling for key structs
func TestTimeEntry_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second) // Truncate for JSON comparison
	stopTime := now.Add(time.Hour)
	projectID := 789

	original := TimeEntry{
		BaseEntity: BaseEntity{
			ID:          1,
			WorkspaceID: 2,
			At:          now,
		},
		ProjectID:   &projectID,
		Description: "Test task",
		Start:       now,
		Stop:        &stopTime,
		Duration:    3600,
		Tags:        []string{"tag1", "tag2"},
		TagIDs:      []int{10, 20},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal TimeEntry: %v", err)
	}

	// Unmarshal back
	var unmarshaled TimeEntry
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal TimeEntry: %v", err)
	}

	// Compare key fields
	if unmarshaled.ID != original.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", unmarshaled.ID, original.ID)
	}
	if unmarshaled.ProjectID == nil || *unmarshaled.ProjectID != *original.ProjectID {
		t.Errorf("Unmarshaled ProjectID = %v, want %v", unmarshaled.ProjectID, original.ProjectID)
	}
	if unmarshaled.Description != original.Description {
		t.Errorf("Unmarshaled Description = %v, want %v", unmarshaled.Description, original.Description)
	}
	if unmarshaled.Duration != original.Duration {
		t.Errorf("Unmarshaled Duration = %v, want %v", unmarshaled.Duration, original.Duration)
	}
}

func TestProject_JSON(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	clientID := 999

	original := Project{
		BaseEntity: BaseEntity{
			ID:          100,
			WorkspaceID: 200,
			At:          now,
		},
		Name:     "Test Project",
		Active:   true,
		Color:    "#FF0000",
		ClientID: &clientID,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Project: %v", err)
	}

	// Unmarshal back
	var unmarshaled Project
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Project: %v", err)
	}

	// Compare key fields
	if unmarshaled.ID != original.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", unmarshaled.ID, original.ID)
	}
	if unmarshaled.Name != original.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", unmarshaled.Name, original.Name)
	}
	if unmarshaled.Active != original.Active {
		t.Errorf("Unmarshaled Active = %v, want %v", unmarshaled.Active, original.Active)
	}
	if unmarshaled.ClientID == nil || *unmarshaled.ClientID != *original.ClientID {
		t.Errorf("Unmarshaled ClientID = %v, want %v", unmarshaled.ClientID, original.ClientID)
	}
}

// Test error wrapping with APIError
func TestAPIError_Unwrap(t *testing.T) {
	apiErr := &APIError{StatusCode: 500, Body: "Server Error"}

	// Test that errors.Is works correctly
	if !errors.Is(apiErr, ErrAPIRequest) {
		t.Errorf("errors.Is(apiErr, ErrAPIRequest) = false, want true")
	}

	// Test that it doesn't match other errors
	if errors.Is(apiErr, ErrNoAPIToken) {
		t.Errorf("errors.Is(apiErr, ErrNoAPIToken) = true, want false")
	}
}
