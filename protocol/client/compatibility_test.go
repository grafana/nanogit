package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/require"
)

func TestIsServerCompatible(t *testing.T) {
	tests := []struct {
		name               string
		responseBody       string
		expectedCompatible bool
		expectError        bool
	}{
		{
			name:               "protocol v2 - version announcement",
			responseBody:       formatTestResponse(t, protocol.PackLine("version 2\n")),
			expectedCompatible: true,
			expectError:        false,
		},
		{
			name:               "protocol v2 - capability line",
			responseBody:       formatTestResponse(t, protocol.PackLine("=capability1\n")),
			expectedCompatible: true,
			expectError:        false,
		},
		{
			name: "protocol v2 - mixed content",
			responseBody: formatTestResponse(t,
				protocol.PackLine("version 2\n"),
				protocol.PackLine("=capability1\n")),
			expectedCompatible: true,
			expectError:        false,
		},
		{
			name: "protocol v1 - ref advertisement",
			// Typical v1 response with ref + capabilities
			responseBody: formatTestResponse(t,
				protocol.PackLine("1234567890abcdef1234567890abcdef12345678 refs/heads/main\000cap1 cap2\n")),
			expectedCompatible: false,
			expectError:        false,
		},
		{
			name: "protocol v1 - multiple refs",
			responseBody: formatTestResponse(t,
				protocol.PackLine("1234567890abcdef1234567890abcdef12345678 refs/heads/main\000cap1\n"),
				protocol.PackLine("abcdef1234567890abcdef1234567890abcdef12 refs/heads/dev\n")),
			expectedCompatible: false,
			expectError:        false,
		},
		{
			name:               "unknown - empty response",
			responseBody:       string(protocol.FlushPacket),
			expectedCompatible: false,
			expectError:        true,
		},
		{
			name:               "unknown - invalid format",
			responseBody:       "invalid data",
			expectedCompatible: false,
			expectError:        true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test the detection function directly
			version := detectProtocolVersionFromReader(strings.NewReader(tt.responseBody))
			expectedVersion := protocolVersionV2
			if !tt.expectedCompatible {
				// Could be v1 or unknown, but we just care if it's not v2
				require.NotEqual(t, protocolVersionV2, version, "protocol version detection mismatch")
			} else {
				require.Equal(t, expectedVersion, version, "protocol version detection mismatch")
			}

			// Test via IsServerCompatible
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if _, err := w.Write([]byte(tt.responseBody)); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := NewRawClient(server.URL + "/repo")
			require.NoError(t, err)

			compatible, err := client.IsServerCompatible(context.Background())
			if tt.expectError {
				require.Error(t, err, "should return error for unknown protocol")
				require.Contains(t, err.Error(), "could not determine protocol version")
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedCompatible, compatible)
		})
	}
}

func TestProtocolVersionDetection_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("hash validation - valid hex", func(t *testing.T) {
		require.True(t, isHexHash([]byte("1234567890abcdefABCDEF1234567890abcdef12")))
	})

	t.Run("hash validation - invalid length", func(t *testing.T) {
		require.False(t, isHexHash([]byte("1234567890abcdef")))                               // too short
		require.False(t, isHexHash([]byte("1234567890abcdef1234567890abcdef12345678901234"))) // too long
	})

	t.Run("hash validation - invalid characters", func(t *testing.T) {
		require.False(t, isHexHash([]byte("1234567890abcdef1234567890abcdef1234567g"))) // 'g' is invalid
		require.False(t, isHexHash([]byte("1234567890abcdef1234567890abcdef1234567 "))) // space is invalid
		require.False(t, isHexHash([]byte("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"))) // all invalid
	})

	t.Run("mixed v1 and v2 indicators - v2 wins", func(t *testing.T) {
		// If a server sends both v1 refs AND v2 indicators, treat as v2
		responseBody := formatTestResponse(t,
			protocol.PackLine("version 2\n"),
			protocol.PackLine("1234567890abcdef1234567890abcdef12345678 refs/heads/main\n"))
		version := detectProtocolVersionFromReader(strings.NewReader(responseBody))
		require.Equal(t, protocolVersionV2, version, "should detect v2 when both indicators present")
	})
}
