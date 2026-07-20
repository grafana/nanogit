package options_test

import (
	"log"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
)

// ExampleWithBasicAuth authenticates with a username and token. Most
// providers accept any username (conventionally "git") with a personal
// access token as the password; GitLab expects "oauth2".
func ExampleWithBasicAuth() {
	client, err := nanogit.NewHTTPClient(
		"https://github.com/owner/repo.git",
		options.WithBasicAuth("git", "your-token"),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

// ExampleWithLimits caps how many bytes the client will read from the server
// per response class, protecting services against oversized or malicious
// responses.
func ExampleWithLimits() {
	client, err := nanogit.NewHTTPClient(
		"https://github.com/owner/repo.git",
		options.WithLimits(options.Limits{
			SingleObjectFetchMaxBytes: 10 << 20,  // 10 MiB per single-object fetch
			MultiObjectFetchMaxBytes:  512 << 20, // 512 MiB per multi-object fetch
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}
