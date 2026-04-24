package options

import (
	"errors"
	"net/http"
	"testing"

	"github.com/grafana/nanogit/protocol"
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

func TestWithReceivePackCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("unset by default", func(t *testing.T) {
		o := &Options{}
		require.Nil(t, o.ReceivePackCapabilities)
	})

	t.Run("replaces the set with what the caller passes", func(t *testing.T) {
		o := &Options{}
		caps := []protocol.Capability{protocol.CapReportStatusV2, protocol.CapAgent("custom")}
		require.NoError(t, WithReceivePackCapabilities(caps...)(o))
		require.Equal(t, caps, o.ReceivePackCapabilities)
	})

	t.Run("copies the slice so caller mutations don't leak", func(t *testing.T) {
		o := &Options{}
		caps := []protocol.Capability{protocol.CapReportStatusV2}
		require.NoError(t, WithReceivePackCapabilities(caps...)(o))
		caps[0] = protocol.CapQuiet
		require.Equal(t, protocol.CapReportStatusV2, o.ReceivePackCapabilities[0])
	})
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
