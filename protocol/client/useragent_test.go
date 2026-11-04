package client

import (
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"testing"

	"github.com/grafana/nanogit/options"
	"github.com/stretchr/testify/require"
)

func TestUserAgentFromBuildInfo(t *testing.T) {
	tests := []struct {
		name string
		bi   *debug.BuildInfo
		want string
	}{
		{
			name: "no build info",
			bi:   nil,
			want: "nanogit",
		},
		{
			name: "nanogit is the main module",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "v1.7.4"},
			},
			want: "nanogit/v1.7.4",
		},
		{
			name: "nanogit is the main module built from a dirty tree",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "v1.7.4+dirty"},
			},
			want: "nanogit/v1.7.4+dirty",
		},
		{
			name: "nanogit is the main module without a version",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: modulePath, Version: "(devel)"},
			},
			want: "nanogit",
		},
		{
			name: "nanogit is a dependency",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app", Version: "(devel)"},
				Deps: []*debug.Module{
					{Path: "github.com/grafana/nanogit/cli", Version: "v0.9.0"},
					{Path: modulePath, Version: "v1.7.3"},
				},
			},
			want: "nanogit/v1.7.3",
		},
		{
			name: "nanogit is a pseudo-version dependency",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app"},
				Deps: []*debug.Module{
					{Path: modulePath, Version: "v1.7.5-0.20261103120000-abcdef123456"},
				},
			},
			want: "nanogit/v1.7.5-0.20261103120000-abcdef123456",
		},
		{
			name: "nanogit dependency replaced by another version",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app"},
				Deps: []*debug.Module{
					{
						Path:    modulePath,
						Version: "v1.7.3",
						Replace: &debug.Module{Path: "github.com/fork/nanogit", Version: "v1.7.4-fork.1"},
					},
				},
			},
			want: "nanogit/v1.7.4-fork.1",
		},
		{
			name: "nanogit dependency replaced by a local path",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app"},
				Deps: []*debug.Module{
					{
						Path:    modulePath,
						Version: "v1.7.3",
						Replace: &debug.Module{Path: "../nanogit", Version: "(devel)"},
					},
				},
			},
			want: "nanogit",
		},
		{
			name: "nanogit dependency from a go.work workspace",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app"},
				Deps: []*debug.Module{
					{Path: modulePath, Version: "(devel)"},
				},
			},
			want: "nanogit",
		},
		{
			name: "nanogit not found in build info",
			bi: &debug.BuildInfo{
				Main: debug.Module{Path: "example.com/app"},
			},
			want: "nanogit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, userAgentFromBuildInfo(tt.bi))
		})
	}
}

func TestDefaultUserAgentHeader(t *testing.T) {
	tests := []struct {
		name string
		opts []options.Option
		want string
	}{
		{
			name: "default",
			opts: nil,
			want: defaultUserAgent(),
		},
		{
			name: "custom",
			opts: []options.Option{options.WithUserAgent("my-app/1.0")},
			want: "my-app/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("User-Agent")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client, err := NewRawClient(server.URL+"/repo", tt.opts...)
			require.NoError(t, err)
			require.NoError(t, client.SmartInfo(t.Context(), "git-upload-pack"))

			require.Equal(t, tt.want, got)
			require.NotEqual(t, "nanogit/0", got)
		})
	}

	bi, ok := debug.ReadBuildInfo()
	require.True(t, ok)
	require.Equal(t, userAgentFromBuildInfo(bi), defaultUserAgent())
	t.Logf("default User-Agent in this test binary: %q (main module %s@%s)", defaultUserAgent(), bi.Main.Path, bi.Main.Version)
}
