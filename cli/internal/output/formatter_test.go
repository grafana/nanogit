package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		expectedType string
	}{
		{
			name:         "json format returns JSONFormatter",
			format:       "json",
			expectedType: "*output.JSONFormatter",
		},
		{
			name:         "empty format returns HumanFormatter",
			format:       "",
			expectedType: "*output.HumanFormatter",
		},
		{
			name:         "unknown format returns HumanFormatter",
			format:       "unknown",
			expectedType: "*output.HumanFormatter",
		},
		{
			name:         "human format returns HumanFormatter",
			format:       "human",
			expectedType: "*output.HumanFormatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := Get(tt.format)
			assert.NotNil(t, formatter)


			// Check the type
			switch tt.expectedType {
			case "*output.JSONFormatter":
				_, ok := formatter.(*JSONFormatter)
				assert.True(t, ok, "expected JSONFormatter")
			case "*output.HumanFormatter":
				_, ok := formatter.(*HumanFormatter)
				assert.True(t, ok, "expected HumanFormatter")
			}
		})
	}
}
