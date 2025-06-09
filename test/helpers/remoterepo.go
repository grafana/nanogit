package helpers

import (
	"fmt"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/stretchr/testify/require"
)

type RemoteRepo struct {
	RepoName string
	User     *User
	Host     string
	Port     string
	logger   *TestLogger
}

func NewRemoteRepo(t *testing.T, logger *TestLogger, repoName string, user *User, host, port string) *RemoteRepo {
	return &RemoteRepo{
		logger:   logger,
		RepoName: repoName,
		User:     user,
		Host:     host,
		Port:     port,
	}
}

func (r *RemoteRepo) URL() string {
	return fmt.Sprintf("http://%s:%s/%s/%s.git", r.Host, r.Port, r.User.Username, r.RepoName)
}

func (r *RemoteRepo) AuthURL() string {
	return fmt.Sprintf("http://%s:%s@%s:%s/%s/%s.git", r.User.Username, r.User.Password, r.Host, r.Port, r.User.Username, r.RepoName)
}

func (r *RemoteRepo) Client(t *testing.T) nanogit.Client {
	client, err := nanogit.NewHTTPClient(r.URL(), nanogit.WithBasicAuth(r.User.Username, r.User.Password), nanogit.WithLogger(r.logger))
	require.NoError(t, err)
	return client
}

func (r *RemoteRepo) Local(t *testing.T) *LocalGitRepo {
	local := NewLocalGitRepo(t, r.logger)
	local.Git(t, "config", "user.name", r.User.Username)
	local.Git(t, "config", "user.email", r.User.Email)
	local.Git(t, "remote", "add", "origin", r.AuthURL())
	return local
}

func (r *RemoteRepo) QuickInit(t *testing.T) (nanogit.Client, *LocalGitRepo) {
	local := NewLocalGitRepo(t, r.logger)
	client, _ := local.QuickInit(t, r.User, r.AuthURL())
	return client, local
}
