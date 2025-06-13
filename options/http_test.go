package options

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithHTTPClient(t *testing.T) {
	tests := []struct {
		name       string
		httpClient *http.Client
		wantErr    error
	}{
		{
			name:       "valid client",
			httpClient: &http.Client{},
			wantErr:    nil,
		},
		{
			name:       "nil client",
			httpClient: nil,
			wantErr:    errors.New("httpClient is nil"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := &Options{}
			err := WithHTTPClient(tt.httpClient)(o)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.httpClient, o.HTTPClient)
		})
	}
}

func TestWithUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      string
	}{
		{
			name:      "custom user agent",
			userAgent: "custom-agent/1.0",
			want:      "custom-agent/1.0",
		},
		{
			name:      "empty user agent",
			userAgent: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := &Options{}
			err := WithUserAgent(tt.userAgent)(o)
			require.NoError(t, err)
			require.Equal(t, tt.want, o.UserAgent)
		})
	}
}
