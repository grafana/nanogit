package helpers

import (
	"fmt"
	"testing"
)

type RemoteRepo struct {
	RepoName string
	Username string
	Password string
	Host     string
	Port     string
}

func NewRemoteRepo(t *testing.T, repoName, username, password, host, port string) *RemoteRepo {
	return &RemoteRepo{
		RepoName: repoName,
		Username: username,
		Password: password,
		Host:     host,
		Port:     port,
	}
}

func (r *RemoteRepo) URL() string {
	return fmt.Sprintf("http://%s:%s/%s/%s.git", r.Host, r.Port, r.Username, r.RepoName)
}

func (r *RemoteRepo) AuthURL() string {
	return fmt.Sprintf("http://%s:%s@%s:%s/%s/%s.git", r.Username, r.Password, r.Host, r.Port, r.Username, r.RepoName)
}
