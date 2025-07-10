package app

import (
	"fmt"
)

// getRequiredNumber extracts a required number parameter
func getRequiredNumber(params map[string]interface{}, key string) (int, error) {
	val, ok := params[key].(float64)
	if !ok {
		return 0, fmt.Errorf("%s must be a number", key)
	}
	return int(val), nil
}

// getOptionalNumber extracts an optional number parameter
func getOptionalNumber(params map[string]interface{}, key string) *int {
	if val, ok := params[key].(float64); ok {
		intVal := int(val)
		return &intVal
	}
	return nil
}

// getRequiredString extracts a required string parameter
func getRequiredString(params map[string]interface{}, key string) (string, error) {
	val, ok := params[key].(string)
	if !ok || val == "" {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}
	return val, nil
}

// getOptionalString extracts an optional string parameter
func getOptionalString(params map[string]interface{}, key string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return ""
}

// formatDuration formats duration in seconds to a human-readable string
func formatDuration(seconds int) string {
	if seconds < 0 {
		return "[running]"
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if hours > 0 {
		return fmt.Sprintf("[%dh %dm %ds]", hours, minutes, secs)
	} else if minutes > 0 {
		return fmt.Sprintf("[%dm %ds]", minutes, secs)
	}
	return fmt.Sprintf("[%ds]", secs)
}
