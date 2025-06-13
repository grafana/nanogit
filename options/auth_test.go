package options

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{
			name:     "valid credentials",
			username: "user",
			password: "pass",
			wantErr:  nil,
		},
		{
			name:     "empty username",
			username: "",
			password: "pass",
			wantErr:  errors.New("username cannot be empty"),
		},
		{
			name:     "empty password allowed",
			username: "user",
			password: "",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := &Options{}
			err := WithBasicAuth(tt.username, tt.password)(o)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.NotNil(t, o.BasicAuth)
			require.Equal(t, tt.username, o.BasicAuth.Username)
			require.Equal(t, tt.password, o.BasicAuth.Password)
		})
	}
}

func TestWithTokenAuth(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "valid token",
			token:   "token123",
			wantErr: nil,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: errors.New("token cannot be empty"),
		},
		{
			name:    "token with bearer prefix",
			token:   "Bearer token123",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := &Options{}
			err := WithTokenAuth(tt.token)(o)
			if tt.wantErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.NotNil(t, o.AuthToken)
			require.Equal(t, tt.token, *o.AuthToken)
		})
	}
}

func TestAuthConflict(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr error
	}{
		{
			name: "basic auth then token auth",
			options: []Option{
				WithBasicAuth("user", "pass"),
				WithTokenAuth("token123"),
			},
			wantErr: errors.New("cannot use both basic auth and token auth"),
		},
		{
			name: "token auth then basic auth",
			options: []Option{
				WithTokenAuth("token123"),
				WithBasicAuth("user", "pass"),
			},
			wantErr: errors.New("cannot use both basic auth and token auth"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := &Options{}
			var err error
			for _, opt := range tt.options {
				err = opt(o)
				if err != nil {
					break
				}
			}
			require.Error(t, err)
			require.Equal(t, tt.wantErr.Error(), err.Error())
		})
	}
}
