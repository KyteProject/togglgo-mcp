package app

import (
	"fmt"
	"testing"
)

func TestGetRequiredNumber(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		key     string
		want    int
		wantErr bool
	}{
		{
			name:    "valid number",
			params:  map[string]interface{}{"count": 42.0},
			key:     "count",
			want:    42,
			wantErr: false,
		},
		{
			name:    "zero value",
			params:  map[string]interface{}{"count": 0.0},
			key:     "count",
			want:    0,
			wantErr: false,
		},
		{
			name:    "negative number",
			params:  map[string]interface{}{"count": -5.0},
			key:     "count",
			want:    -5,
			wantErr: false,
		},
		{
			name:    "fractional number truncated",
			params:  map[string]interface{}{"count": 42.7},
			key:     "count",
			want:    42,
			wantErr: false,
		},
		{
			name:    "missing key",
			params:  map[string]interface{}{"other": 42.0},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type - string",
			params:  map[string]interface{}{"count": "42"},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "wrong type - int",
			params:  map[string]interface{}{"count": 42},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "nil value",
			params:  map[string]interface{}{"count": nil},
			key:     "count",
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty params",
			params:  map[string]interface{}{},
			key:     "count",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRequiredNumber(tt.params, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRequiredNumber() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getRequiredNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOptionalNumber(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		key    string
		want   *int
	}{
		{
			name:   "valid number",
			params: map[string]interface{}{"count": 42.0},
			key:    "count",
			want:   intPtr(42),
		},
		{
			name:   "zero value",
			params: map[string]interface{}{"count": 0.0},
			key:    "count",
			want:   intPtr(0),
		},
		{
			name:   "negative number",
			params: map[string]interface{}{"count": -5.0},
			key:    "count",
			want:   intPtr(-5),
		},
		{
			name:   "fractional number truncated",
			params: map[string]interface{}{"count": 42.7},
			key:    "count",
			want:   intPtr(42),
		},
		{
			name:   "missing key",
			params: map[string]interface{}{"other": 42.0},
			key:    "count",
			want:   nil,
		},
		{
			name:   "wrong type - string",
			params: map[string]interface{}{"count": "42"},
			key:    "count",
			want:   nil,
		},
		{
			name:   "wrong type - int",
			params: map[string]interface{}{"count": 42},
			key:    "count",
			want:   nil,
		},
		{
			name:   "nil value",
			params: map[string]interface{}{"count": nil},
			key:    "count",
			want:   nil,
		},
		{
			name:   "empty params",
			params: map[string]interface{}{},
			key:    "count",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOptionalNumber(tt.params, tt.key)
			if !equalIntPtr(got, tt.want) {
				t.Errorf("getOptionalNumber() = %v, want %v", ptrToString(got), ptrToString(tt.want))
			}
		})
	}
}

func TestGetRequiredString(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		key     string
		want    string
		wantErr bool
	}{
		{
			name:    "valid string",
			params:  map[string]interface{}{"name": "test"},
			key:     "name",
			want:    "test",
			wantErr: false,
		},
		{
			name:    "string with spaces",
			params:  map[string]interface{}{"name": "  test  "},
			key:     "name",
			want:    "  test  ",
			wantErr: false,
		},
		{
			name:    "single character",
			params:  map[string]interface{}{"name": "a"},
			key:     "name",
			want:    "a",
			wantErr: false,
		},
		{
			name:    "empty string",
			params:  map[string]interface{}{"name": ""},
			key:     "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing key",
			params:  map[string]interface{}{"other": "test"},
			key:     "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type - number",
			params:  map[string]interface{}{"name": 42.0},
			key:     "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "wrong type - bool",
			params:  map[string]interface{}{"name": true},
			key:     "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "nil value",
			params:  map[string]interface{}{"name": nil},
			key:     "name",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty params",
			params:  map[string]interface{}{},
			key:     "name",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRequiredString(tt.params, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRequiredString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getRequiredString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOptionalString(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		key    string
		want   string
	}{
		{
			name:   "valid string",
			params: map[string]interface{}{"name": "test"},
			key:    "name",
			want:   "test",
		},
		{
			name:   "empty string",
			params: map[string]interface{}{"name": ""},
			key:    "name",
			want:   "",
		},
		{
			name:   "string with spaces",
			params: map[string]interface{}{"name": "  test  "},
			key:    "name",
			want:   "  test  ",
		},
		{
			name:   "single character",
			params: map[string]interface{}{"name": "a"},
			key:    "name",
			want:   "a",
		},
		{
			name:   "missing key",
			params: map[string]interface{}{"other": "test"},
			key:    "name",
			want:   "",
		},
		{
			name:   "wrong type - number",
			params: map[string]interface{}{"name": 42.0},
			key:    "name",
			want:   "",
		},
		{
			name:   "wrong type - bool",
			params: map[string]interface{}{"name": true},
			key:    "name",
			want:   "",
		},
		{
			name:   "nil value",
			params: map[string]interface{}{"name": nil},
			key:    "name",
			want:   "",
		},
		{
			name:   "empty params",
			params: map[string]interface{}{},
			key:    "name",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOptionalString(tt.params, tt.key)
			if got != tt.want {
				t.Errorf("getOptionalString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{
			name:    "negative duration (running)",
			seconds: -1,
			want:    "[running]",
		},
		{
			name:    "zero seconds",
			seconds: 0,
			want:    "[0s]",
		},
		{
			name:    "single digit seconds",
			seconds: 5,
			want:    "[5s]",
		},
		{
			name:    "double digit seconds",
			seconds: 59,
			want:    "[59s]",
		},
		{
			name:    "exactly one minute",
			seconds: 60,
			want:    "[1m 0s]",
		},
		{
			name:    "minutes and seconds",
			seconds: 125,
			want:    "[2m 5s]",
		},
		{
			name:    "exactly one hour",
			seconds: 3600,
			want:    "[1h 0m 0s]",
		},
		{
			name:    "hours only",
			seconds: 7200,
			want:    "[2h 0m 0s]",
		},
		{
			name:    "hours and minutes",
			seconds: 3660,
			want:    "[1h 1m 0s]",
		},
		{
			name:    "hours and seconds",
			seconds: 3605,
			want:    "[1h 0m 5s]",
		},
		{
			name:    "hours, minutes, and seconds",
			seconds: 3665,
			want:    "[1h 1m 5s]",
		},
		{
			name:    "large duration",
			seconds: 90061,
			want:    "[25h 1m 1s]",
		},
		{
			name:    "maximum edge case",
			seconds: 359999,
			want:    "[99h 59m 59s]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.seconds)
			if got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions for testing
func intPtr(i int) *int {
	return &i
}

func equalIntPtr(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrToString(p *int) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *p)
}
