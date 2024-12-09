package client

import (
	"time"

	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
)

// A Commit entails a set of data and an object.
type Commit struct {
	// Hash contains the hash of the commit itself. This is the full hash, not a short one.
	Hash string `json:"hash"`
	// The parent is the base commit on which this commit builds.
	Parent string `json:"parent"`
	// The author created this commit. Note that the message could contain more authors and similar.
	Author string `json:"author"`
	// The committer last touched this commit in re-creating it.
	Committer string `json:"committer"`
	// The timestamp denotes when this commit was created.
	Timestamp time.Time `json:"timestamp"`
	// The message contains a synopsis of the commit.
	// It may also include a PGP signature.
	Message string `json:"message"`

	// AdditionalFields are fields beyond what we know to parse.
	// This is not a stable field: if we model more fields, they may disappear here.
	// If a field is statically defined, it SHOULD not show up here.
	AdditionalFields map[string][]byte `json:"additional_fields"`
}

type FileHistory struct {
	base   []byte
	deltas []fileDelta
}

type fileDelta struct {
	// The commit to which this delta is associated.
	commit string
	// The delta that this entails, including the parent commit.
	delta protocol.Delta
}

// Ideal API:
//   client.Fetch(ctx, "HEAD" [or other ref]) (*History, error)
//   client.Snapshot(ctx, history, "file/path", commitRef [optional?]) (*File, error)
// History needs at least the base commit (ie earliest commit) + its base data. And deltas on each commit since.

type History struct {
	Ref     string `json:"ref"`
	Commits []Commit
}
