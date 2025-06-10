package helpers

import (
	"fmt"

	"github.com/grafana/nanogit"
	. "github.com/onsi/gomega"
)

type RemoteRepo struct {
	RepoName string
	User     *User
	Host     string
	Port     string
	logger   *TestLogger
}

func NewRemoteRepo(logger *TestLogger, repoName string, user *User, host, port string) *RemoteRepo {
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

func (r *RemoteRepo) Client() nanogit.Client {
	client, err := nanogit.NewHTTPClient(r.URL(), nanogit.WithBasicAuth(r.User.Username, r.User.Password), nanogit.WithLogger(r.logger))
	Expect(err).NotTo(HaveOccurred())
	return client
}

func (r *RemoteRepo) Local() *LocalGitRepo {
	local := NewLocalGitRepo(r.logger)
	local.Git("config", "user.name", r.User.Username)
	local.Git("config", "user.email", r.User.Email)
	local.Git("remote", "add", "origin", r.AuthURL())
	return local
}

func (r *RemoteRepo) QuickInit() (nanogit.Client, *LocalGitRepo) {
	local := NewLocalGitRepo(r.logger)
	client, _ := local.QuickInit(r.User, r.AuthURL())
	return client, local
}
