package app

import (
	"errors"
	"fmt"
	"time"
)

// Custom error types for better error handling
var (
	ErrNoAPIToken       = errors.New("TOGGL_API_TOKEN environment variable is required")
	ErrInvalidWorkspace = errors.New("workspace_id must be a number")
	ErrInvalidProject   = errors.New("project_id must be a number")
	ErrInvalidDate      = errors.New("invalid date format")
	ErrNoRunningEntry   = errors.New("no running time entry found")
	ErrAPIRequest       = errors.New("API request failed")
)

// APIError represents an error from the Toggl API
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

func (e *APIError) Is(target error) bool {
	return target == ErrAPIRequest
}

// BaseEntity contains common fields for Toggl entities
type BaseEntity struct {
	ID          int       `json:"id"`
	WorkspaceID int       `json:"workspace_id"`
	At          time.Time `json:"at"`
}

// TimeEntry represents a Toggl time entry
type TimeEntry struct {
	BaseEntity
	ProjectID   *int       `json:"project_id,omitempty"`
	Description string     `json:"description,omitempty"`
	Start       time.Time  `json:"start"`
	Stop        *time.Time `json:"stop,omitempty"`
	Duration    int        `json:"duration"`
	Tags        []string   `json:"tags,omitempty"`
	TagIDs      []int      `json:"tag_ids,omitempty"`
}

// Project represents a Toggl project
type Project struct {
	BaseEntity
	Name     string `json:"name"`
	Active   bool   `json:"active"`
	Color    string `json:"color,omitempty"`
	ClientID *int   `json:"client_id,omitempty"`
}

// TimeEntryRequest represents the payload for creating a time entry
type TimeEntryRequest struct {
	Description string    `json:"description"`
	Start       time.Time `json:"start"`
	Duration    int       `json:"duration"`
	ProjectID   *int      `json:"project_id,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedWith string    `json:"created_with"`
}

// UserInfo represents user account information
type UserInfo struct {
	ID                 int    `json:"id"`
	Email              string `json:"email"`
	Fullname           string `json:"fullname"`
	DefaultWorkspaceID int    `json:"default_workspace_id"`
}
