package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// SetupTools defines all tools with their configurations
func SetupTools(s *server.MCPServer, togglClient *TogglClient) error {
	tools := []struct {
		tool    mcp.Tool
		handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{
			tool: mcp.NewTool(
				"test_connection",
				mcp.WithDescription("Test the Toggl API connection and authentication"),
			),
			handler: wrapHandler(togglClient, handleTestConnection),
		},
		{
			tool: mcp.NewTool(
				"start_time_entry",
				mcp.WithDescription("Start a new time entry"),
				mcp.WithString("description", mcp.Required()),
				mcp.WithNumber("workspace_id", mcp.Required()),
				mcp.WithNumber("project_id"),
			),
			handler: wrapHandler(togglClient, handleStartTimeEntry),
		},
		{
			tool: mcp.NewTool(
				"stop_time_entry",
				mcp.WithDescription("Stop the current running time entry"),
				mcp.WithNumber("workspace_id", mcp.Required()),
			),
			handler: wrapHandler(togglClient, handleStopTimeEntry),
		},
		{
			tool: mcp.NewTool(
				"get_current_time_entry",
				mcp.WithDescription("Get the currently running time entry"),
			),
			handler: wrapHandler(togglClient, handleGetCurrentTimeEntry),
		},
		{
			tool: mcp.NewTool(
				"get_time_entries",
				mcp.WithDescription("Get time entries with optional date filtering. Note: To get entries for a single day (e.g., July 9th), use start_date=2025-07-09 and end_date=2025-07-10. The API uses inclusive start, exclusive end date logic."),
				mcp.WithString("start_date"),
				mcp.WithString("end_date"),
			),
			handler: wrapHandler(togglClient, handleGetTimeEntries),
		},
		{
			tool: mcp.NewTool(
				"get_time_entries_for_day",
				mcp.WithDescription("Get time entries for a specific day (automatically handles the date range correctly)"),
				mcp.WithString("date", mcp.Required()),
			),
			handler: wrapHandler(togglClient, handleGetTimeEntriesForDay),
		},
		{
			tool: mcp.NewTool(
				"create_project",
				mcp.WithDescription("Create a new project"),
				mcp.WithString("name", mcp.Required()),
				mcp.WithNumber("workspace_id", mcp.Required()),
				mcp.WithString("color"),
				mcp.WithNumber("client_id"),
			),
			handler: wrapHandler(togglClient, handleCreateProject),
		},
		{
			tool: mcp.NewTool(
				"get_projects",
				mcp.WithDescription("Get projects in a workspace"),
				mcp.WithNumber("workspace_id", mcp.Required()),
				mcp.WithBoolean("active"),
			),
			handler: wrapHandler(togglClient, handleGetProjects),
		},
	}

	// Register all tools
	for _, t := range tools {
		s.AddTool(t.tool, t.handler)
	}

	return nil
}

// wrapHandler wraps a handler function to provide the client and proper error handling
func wrapHandler(
	client *TogglClient,
	handler func(
		context.Context,
		*TogglClient,
		mcp.CallToolRequest,
	) (*mcp.CallToolResult, error),
) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handler(ctx, client, req)
	}
}

func handleTestConnection(
	ctx context.Context,
	client *TogglClient,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	resp, err := client.makeRequest(ctx, http.MethodGet, "/me", nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to Toggl API: %w", err)
	}

	user, err := decodeResponse[UserInfo](resp)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return mcp.NewToolResultError(fmt.Sprintf("Authentication failed: %s", apiErr.Error())), nil
		}
		return nil, fmt.Errorf("parsing user info: %w", err)
	}

	result := fmt.Sprintf(`✅ Authentication successful!
User: %s
Email: %s
Default Workspace ID: %d`, user.Fullname, user.Email, user.DefaultWorkspaceID)

	return mcp.NewToolResultText(result), nil
}

func handleStartTimeEntry(
	ctx context.Context,
	client *TogglClient,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	description, err := getRequiredString(req.Params.Arguments, "description")
	if err != nil {
		return nil, fmt.Errorf("invalid description: %w", err)
	}

	workspaceID, err := getRequiredNumber(req.Params.Arguments, "workspace_id")
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	entry := TimeEntryRequest{
		Description: description,
		Start:       time.Now(),
		Duration:    -1, // Running timer
		CreatedWith: "toggl-mcp",
		ProjectID:   getOptionalNumber(req.Params.Arguments, "project_id"),
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := client.makeRequest(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/workspaces/%d/time_entries", workspaceID),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, fmt.Errorf("starting time entry: %w", err)
	}

	result, err := decodeResponse[TimeEntry](resp)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Failed to start time entry: %s", apiErr.Error()),
			), nil
		}
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Started time entry: %s (ID: %d)", result.Description, result.ID),
	), nil
}

func handleStopTimeEntry(
	ctx context.Context,
	client *TogglClient,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	workspaceID, err := getRequiredNumber(req.Params.Arguments, "workspace_id")
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	// Get current running entry
	resp, err := client.makeRequest(ctx, http.MethodGet, "/me/time_entries/current", nil)
	if err != nil {
		return nil, fmt.Errorf("getting current time entry: %w", err)
	}

	// Check if there's a running entry
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return mcp.NewToolResultText("No running time entry found"), nil
	}

	current, err := decodeResponse[TimeEntry](resp)
	if err != nil {
		return nil, fmt.Errorf("parsing current entry: %w", err)
	}

	// Stop the entry
	stopResp, err := client.makeRequest(
		ctx,
		http.MethodPatch,
		fmt.Sprintf("/workspaces/%d/time_entries/%d/stop", workspaceID, current.ID),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("stopping time entry: %w", err)
	}

	_, err = decodeResponse[TimeEntry](stopResp)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Failed to stop time entry: %s", apiErr.Error()),
			), nil
		}
		return nil, fmt.Errorf("decoding stop response: %w", err)
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Stopped time entry: %s (ID: %d)", current.Description, current.ID),
	), nil
}

func handleGetCurrentTimeEntry(
	ctx context.Context,
	client *TogglClient,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	resp, err := client.makeRequest(ctx, http.MethodGet, "/me/time_entries/current", nil)
	if err != nil {
		return nil, fmt.Errorf("getting current time entry: %w", err)
	}

	// Check if there's a running entry
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return mcp.NewToolResultText("No running time entry found"), nil
	}

	current, err := decodeResponse[TimeEntry](resp)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Failed to get current entry: %s", apiErr.Error()),
			), nil
		}
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	duration := time.Since(current.Start).Round(time.Second)
	return mcp.NewToolResultText(fmt.Sprintf("Current time entry: %s (ID: %d, Running for: %s)",
		current.Description, current.ID, duration)), nil
}

func handleGetTimeEntries(
	ctx context.Context,
	client *TogglClient,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	endpoint := "/me/time_entries"
	params := url.Values{}

	startDate := getOptionalString(req.Params.Arguments, "start_date")
	endDate := getOptionalString(req.Params.Arguments, "end_date")

	if startDate != "" {
		if _, err := time.Parse("2006-01-02", startDate); err != nil {
			return nil, fmt.Errorf("invalid start_date format (use YYYY-MM-DD): %w", err)
		}
		params.Set("start_date", startDate)
	}
	if endDate != "" {
		if _, err := time.Parse("2006-01-02", endDate); err != nil {
			return nil, fmt.Errorf("invalid end_date format (use YYYY-MM-DD): %w", err)
		}
		params.Set("end_date", endDate)
	}

	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := client.makeRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting time entries: %w", err)
	}

	entries, err := decodeResponse[[]TimeEntry](resp)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Failed to get time entries: %s", apiErr.Error()),
			), nil
		}
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d time entries:\n", len(entries)))

	if len(entries) == 0 {
		result.WriteString("\nNo time entries found. This could be because:\n")
		result.WriteString("- No time entries exist in the specified date range\n")
		result.WriteString("- You need to specify a date range (start_date, end_date)\n")
		result.WriteString("- The default query only returns recent entries\n")

		// Check if user tried to use same date for start and end
		if startDate != "" && endDate != "" && startDate == endDate {
			result.WriteString("\n⚠️  LIKELY ISSUE: You used the same date for start_date and end_date!\n")
			result.WriteString(fmt.Sprintf("To get entries for %s, use:\n", startDate))
			result.WriteString(fmt.Sprintf("  start_date: %s\n", startDate))

			// Calculate next day
			if date, err := time.Parse("2006-01-02", startDate); err == nil {
				nextDay := date.AddDate(0, 0, 1).Format("2006-01-02")
				result.WriteString(fmt.Sprintf("  end_date: %s\n", nextDay))
				result.WriteString(fmt.Sprintf("OR use the 'get_time_entries_for_day' tool with date: %s\n", startDate))
			}
		}
	} else {
		for _, entry := range entries {
			duration := formatDuration(entry.Duration)
			projectInfo := ""
			if entry.ProjectID != nil {
				projectInfo = fmt.Sprintf(" (Project ID: %d)", *entry.ProjectID)
			}
			result.WriteString(fmt.Sprintf("- %s (ID: %d)%s %s\n",
				entry.Description, entry.ID, projectInfo, duration))
		}
	}

	return mcp.NewToolResultText(result.String()), nil
}

func handleGetTimeEntriesForDay(
	ctx context.Context,
	client *TogglClient,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	date, err := getRequiredString(req.Params.Arguments, "date")
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}

	startDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
	}

	endDate := startDate.AddDate(0, 0, 1) // Add one day
	endDateStr := endDate.Format("2006-01-02")

	req.Params.Arguments["start_date"] = date
	req.Params.Arguments["end_date"] = endDateStr

	return handleGetTimeEntries(ctx, client, req)
}

func handleCreateProject(
	ctx context.Context,
	client *TogglClient,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, err := getRequiredString(req.Params.Arguments, "name")
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}

	workspaceID, err := getRequiredNumber(req.Params.Arguments, "workspace_id")
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	project := map[string]interface{}{
		"name":   name,
		"active": true,
	}

	if color := getOptionalString(req.Params.Arguments, "color"); color != "" {
		project["color"] = color
	}
	if clientID := getOptionalNumber(req.Params.Arguments, "client_id"); clientID != nil {
		project["client_id"] = *clientID
	}

	payload, err := json.Marshal(project)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	resp, err := client.makeRequest(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/workspaces/%d/projects", workspaceID),
		strings.NewReader(string(payload)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}

	result, err := decodeResponse[Project](resp)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Failed to create project: %s", apiErr.Error()),
			), nil
		}
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Created project: %s (ID: %d)", result.Name, result.ID),
	), nil
}

func handleGetProjects(
	ctx context.Context,
	client *TogglClient,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	workspaceID, err := getRequiredNumber(req.Params.Arguments, "workspace_id")
	if err != nil {
		return nil, fmt.Errorf("invalid workspace_id: %w", err)
	}

	endpoint := fmt.Sprintf("/workspaces/%d/projects", workspaceID)
	params := url.Values{}

	if active, ok := req.Params.Arguments["active"].(bool); ok {
		params.Set("active", strconv.FormatBool(active))
	}

	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	resp, err := client.makeRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getting projects: %w", err)
	}

	projects, err := decodeResponse[[]Project](resp)
	if err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			return mcp.NewToolResultError(
				fmt.Sprintf("Failed to get projects: %s", apiErr.Error()),
			), nil
		}
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d projects:\n", len(projects)))

	for _, project := range projects {
		status := "inactive"
		if project.Active {
			status = "active"
		}
		result.WriteString(
			fmt.Sprintf("- %s (ID: %d, %s)\n", project.Name, project.ID, status),
		)
	}

	return mcp.NewToolResultText(result.String()), nil
}
