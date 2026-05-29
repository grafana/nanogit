package client

import (
	"context"
	"errors"
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
			version, err := detectProtocolVersionFromReader(strings.NewReader(tt.responseBody), 0)
			require.NoError(t, err)
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
		version, err := detectProtocolVersionFromReader(strings.NewReader(responseBody), 0)
		require.NoError(t, err)
		require.Equal(t, protocolVersionV2, version, "should detect v2 when both indicators present")
	})
}

func TestDetectProtocolVersionFromReaderSurfacesLimitErr(t *testing.T) {
	t.Parallel()

	// Body is much larger than the cap AND has no v1/v2 indicator, so
	// the limit reader trips and the function must surface
	// *ErrResponseTooLarge to the caller. Without the typed error, an
	// oversize info/refs response with no decisive bytes would silently
	// look like an incompatible server — exactly the failure mode
	// operators tuning RefsMetadataMaxBytes need to distinguish.
	bigBody := strings.Repeat("x", 4096)
	_, err := detectProtocolVersionFromReader(strings.NewReader(bigBody), 64)
	require.Error(t, err)

	var tooLarge *ErrResponseTooLarge
	require.True(t, errors.As(err, &tooLarge),
		"expected *ErrResponseTooLarge, got %T: %v", err, err)
	require.Equal(t, "compatibility", tooLarge.Op)
	require.Equal(t, int64(64), tooLarge.Limit)
}

func TestDetectProtocolVersionV1DespiteCapHit(t *testing.T) {
	t.Parallel()

	// A large v1 response (many refs) must still be identified as v1
	// even when the cap fires before we finish reading the body. The
	// cap is only there to bound memory; once we have a v1 ref line
	// in hand the additional bytes are just more refs of the same
	// kind, so surfacing *ErrResponseTooLarge here would regress the
	// documented "compatible=false" path for genuinely large v1 repos.
	body := formatTestResponse(t,
		// First a real v1 ref line that fits inside the cap.
		protocol.PackLine("1234567890abcdef1234567890abcdef12345678 refs/heads/main\n"),
	)
	// Pad with additional bytes well past the cap so io.ReadAll
	// returns *ErrResponseTooLarge alongside the partial buffer.
	body += strings.Repeat("x", 8192)

	version, err := detectProtocolVersionFromReader(strings.NewReader(body), 128)
	require.NoError(t, err, "v1 must be identifiable from the bytes that fit in the cap")
	require.Equal(t, protocolVersionV1, version)
}

func TestCompatibilityReadLimit(t *testing.T) {
	t.Parallel()

	t.Run("zero (unlimited) returns the floor", func(t *testing.T) {
		require.Equal(t, int64(compatibilityFloor), compatibilityReadLimit(0))
	})

	t.Run("negative returns the floor", func(t *testing.T) {
		require.Equal(t, int64(compatibilityFloor), compatibilityReadLimit(-1))
	})

	t.Run("smaller than floor is honored as configured", func(t *testing.T) {
		// An explicit positive value is the operator telling us what
		// they want — even if it is below the unlimited-fallback
		// floor. The floor only applies when the caller leaves
		// RefsMetadataMaxBytes at zero (unlimited).
		require.Equal(t, int64(1024), compatibilityReadLimit(1024))
	})

	t.Run("larger than floor passes through", func(t *testing.T) {
		require.Equal(t, int64(2*compatibilityFloor), compatibilityReadLimit(2*compatibilityFloor))
	})
}
