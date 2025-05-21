package nanogit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuthentication(t *testing.T) {
	tests := []struct {
		name           string
		authOption     Option
		expectedHeader string
	}{
		{
			name:           "basic auth",
			authOption:     WithBasicAuth("user", "pass"),
			expectedHeader: "Basic dXNlcjpwYXNz",
		},
		{
			name:           "token auth",
			authOption:     WithTokenAuth("token123"),
			expectedHeader: "token123",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check default headers
				if gitProtocol := r.Header.Get("Git-Protocol"); gitProtocol != "version=2" {
					t.Errorf("expected Git-Protocol header 'version=2', got %s", gitProtocol)
					return
				}
				if userAgent := r.Header.Get("User-Agent"); userAgent != "nanogit/0" {
					t.Errorf("expected User-Agent header 'nanogit/0', got %s", userAgent)
					return
				}

				auth := r.Header.Get("Authorization")
				if auth != tt.expectedHeader {
					t.Errorf("expected Authorization header %s, got %s", tt.expectedHeader, auth)
					return
				}

				if contentType := r.Header.Get("Content-Type"); contentType != "application/x-git-upload-pack-request" {
					t.Errorf("expected Content-Type header 'application/x-git-upload-pack-request', got %s", contentType)
					return
				}

				if _, err := w.Write([]byte("ok")); err != nil {
					t.Errorf("failed to write response: %v", err)
					return
				}
			}))
			defer server.Close()

			client, err := NewClient(server.URL, tt.authOption)
			require.NoError(t, err)

			c, ok := client.(*clientImpl)
			require.True(t, ok, "client should be of type *client")

			_, err = c.uploadPack(context.Background(), []byte("test"))
			require.NoError(t, err)
		})
	}
}
