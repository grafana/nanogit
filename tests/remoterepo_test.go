package nanogit_test

import "fmt"

type RemoteRepo struct {
	RepoName string
	User     *User
	Host     string
	Port     string
}

func NewRemoteRepo(repoName string, user *User, host, port string) *RemoteRepo {
	return &RemoteRepo{
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
