package protocol_test

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
)

func TestParseRefName(t *testing.T) {
	t.Parallel()

	t.Run("HEAD is valid", func(t *testing.T) {
		// git check-ref-format does not consider HEAD to be valid,
		// make a special case for it.
		refname, err := protocol.ParseRefName("HEAD")
		require.NoError(t, err, "parsing HEAD should succeed")
		require.Equal(t, protocol.HEAD, refname, "parsed refname should be HEAD")
	})

	t.Run("parse valid ref names", func(t *testing.T) {
		testcases := []struct {
			Full     string
			Category string
			Location string
		}{
			{"refs/heads/main", "heads", "main"},
			{"refs/heads/feature/test", "heads", "feature/test"},
			{"refs/heads/foo./bar", "heads", "foo./bar"},
		}

		for _, tc := range testcases {
			t.Run("parse: "+tc.Full, func(t *testing.T) {
				require.Truef(t, validateWithGitCheckRefFormat(t, tc.Full), "git check-ref-format considers %q to be valid", tc.Full)
				rn, err := protocol.ParseRefName(tc.Full)
				require.NoError(t, err, "expected parsing valid refname to succeed, but it failed")
				require.Equal(t, protocol.RefName{
					FullName: tc.Full,
					Category: tc.Category,
					Location: tc.Location,
				}, rn)
			})
		}
	})

	t.Run("parse invalid ref names", func(t *testing.T) {
		testcases := []struct {
			Value string
			Name  string
		}{
			{"", "empty"},
			{"@", "references cannot be the single character @"},
			{"H", "single H character"},
			{"refs/", "only refs prefix"},
			{"refs//", "all empty"},
			{"refs//test", "empty category"},
			{"refs/../test", ".. category"},
			{"refs/heads/.bar", "no slash-separated component can begin with a dot ."},
			{"refs/heads/foo.lock", "no slash-separated component can end with the sequence .lock."},
			{"refs/heads/foo.lock/bar", "no slash-separated component can end with the sequence .lock."},
			{"refs/heads/.lock", "no slash-separated component can end with the sequence .lock."},
			{"refs/heads/foo..bar", "references cannot have two consecutive dots .. anywhere."},
			{"refs/heads/foo\001bar", "references cannot have control characters."},
			{"refs/heads/foo\002bar", "references cannot have control characters."},
			{"refs/heads/foo\003bar", "references cannot have control characters."},
			{"refs/heads/foo\004bar", "references cannot have control characters."},
			{"refs/heads/foo\005bar", "references cannot have control characters."},
			{"refs/heads/foo\006bar", "references cannot have control characters."},
			{"refs/heads/foo\007bar", "references cannot have control characters."},
			{"refs/heads/foo\010bar", "references cannot have control characters."},
			{"refs/heads/foo\011bar", "references cannot have control characters."},
			{"refs/heads/foo\012bar", "references cannot have control characters."},
			{"refs/heads/foo\013bar", "references cannot have control characters."},
			{"refs/heads/foo\014bar", "references cannot have control characters."},
			{"refs/heads/foo\015bar", "references cannot have control characters."},
			{"refs/heads/foo\016bar", "references cannot have control characters."},
			{"refs/heads/foo\017bar", "references cannot have control characters."},
			{"refs/heads/foo\020bar", "references cannot have control characters."},
			{"refs/heads/foo\021bar", "references cannot have control characters."},
			{"refs/heads/foo\022bar", "references cannot have control characters."},
			{"refs/heads/foo\023bar", "references cannot have control characters."},
			{"refs/heads/foo\024bar", "references cannot have control characters."},
			{"refs/heads/foo\025bar", "references cannot have control characters."},
			{"refs/heads/foo\026bar", "references cannot have control characters."},
			{"refs/heads/foo\027bar", "references cannot have control characters."},
			{"refs/heads/foo\030bar", "references cannot have control characters."},
			{"refs/heads/foo\031bar", "references cannot have control characters."},
			{"refs/heads/foo\032bar", "references cannot have control characters."},
			{"refs/heads/foo\033bar", "references cannot have control characters."},
			{"refs/heads/foo\034bar", "references cannot have control characters."},
			{"refs/heads/foo\035bar", "references cannot have control characters."},
			{"refs/heads/foo\036bar", "references cannot have control characters."},
			{"refs/heads/foo\037bar", "references cannot have control characters."},
			{"refs/heads/foo\040bar", "references cannot have control characters."},
			{"refs/heads/foo\177bar", "references cannot have control characters."},
			{"refs/heads/foo bar", "references cannot have space anywhere."},
			{"refs/heads/foo~bar", "references cannot have tilde anywhere."},
			{"refs/heads/foo^bar", "references cannot have caret anywhere."},
			{"refs/heads/foo:bar", "references cannot have colon anywhere."},
			{"refs/heads/foo?bar", "references cannot have question-mark anywhere."},
			{"refs/heads/foo*bar", "references cannot have asterisk anywhere."},
			{"refs/heads/foo[bar", "references cannot have open bracket anywhere."},
			{"refs/heads/foobar/", "references cannot end with a slash."},
			{"refs/heads/foo//bar", "references cannot contain multiple consecutive slashes."},
			{"refs/heads/foobar.", "references cannot end with a dot."},
			{"refs/heads/foo@{bar.", "references cannot contain the sequence @{."},
			{"refs/.heads/test", "category starting with ."},
			{"refs/he..ads/test", "otherwise valid category containing with .."},
			{"refs/heads@{1}/", "otherwise valid category containing @{"},
			{"refs/heads\\\\/", "otherwise valid category containing \\\\"},
			{"refs/hea ds/test", "otherwise valid category containing a space"},
			{"refs/hea:ds/test", "otherwise valid category containing a colon"},
			{"refs/hea?ds/test", "otherwise valid category containing a question mark"},
			{"refs/hea*ds/test", "otherwise valid category containing an asterisk"},
			{"refs/hea[ds/test", "otherwise valid category containing an open square bracket"},
			{"refs/heads\177/test", "otherwise valid category containing a DEL"},
			{"refs/heads\033/test", "otherwise valid category containing a byte < 40"},
			{"refs/heads/test/", "otherwise valid refname ending with a slash"},
		}

		for _, tc := range testcases {
			t.Run("parse: "+tc.Name, func(t *testing.T) {
				require.Falsef(t, validateWithGitCheckRefFormat(t, tc.Value), "git check-ref-format considers %q to be invalid", tc.Value)
				_, err := protocol.ParseRefName(tc.Value)
				require.Error(t, err, `parsing refname "%q" should fail`, tc.Value)
			})
		}
	})

	// Special cases.

	t.Run("references should not contain NUL", func(t *testing.T) {
		// A NUL byte cannot be passed as an argument to git check-ref-format.
		_, err := protocol.ParseRefName("refs/heads/foo\000bar")
		require.Error(t, err, "parsing refname with NUL byte should fail")
	})
}

func validateWithGitCheckRefFormat(t *testing.T, refName string) bool {
	t.Helper()

	stderr := &bytes.Buffer{}
	stdout := &bytes.Buffer{}

	cmd := exec.Command("git", "check-ref-format", refName)
	cmd.Stderr = stderr
	cmd.Stdout = stdout

	err := cmd.Run()

	if stdout.Len() > 0 {
		t.Logf("stdout: %s", stdout.String())
	}

	if stderr.Len() > 0 {
		t.Logf("stderr: %s", stderr.String())
	}

	if err != nil {
		var execErr *exec.ExitError
		if !errors.As(err, &execErr) {
			t.Fatalf("failed to run git check-ref-format: %v\nstderr: %s", err, stderr.String())
		}
	}

	return cmd.ProcessState.ExitCode() == 0
}

func TestRefUpdateRequest_Format(t *testing.T) {
	tests := []struct {
		name    string
		req     protocol.RefUpdateRequest
		wantPkt string
		wantErr bool
		errMsg  string
	}{
		{
			name: "create ref",
			req: protocol.RefUpdateRequest{
				OldRef:  protocol.ZeroHash,
				NewRef:  "1234567890123456789012345678901234567890",
				RefName: "refs/heads/main",
			},
			wantPkt: "0000000000000000000000000000000000000000 1234567890123456789012345678901234567890 refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		},
		{
			name: "update ref",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  "abcdef0123456789abcdef0123456789abcdef01",
				RefName: "refs/heads/main",
			},
			wantPkt: "1234567890123456789012345678901234567890 abcdef0123456789abcdef0123456789abcdef01 refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		},
		{
			name: "delete ref",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  protocol.ZeroHash,
				RefName: "refs/heads/main",
			},
			wantPkt: "1234567890123456789012345678901234567890 0000000000000000000000000000000000000000 refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		},
		{
			name: "invalid old ref hash length",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234", // too short
				NewRef:  "1234567890123456789012345678901234567890",
				RefName: "refs/heads/main",
			},
			wantErr: true,
			errMsg:  "invalid old ref hash length",
		},
		{
			name: "invalid new ref hash length",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  "1234", // too short
				RefName: "refs/heads/main",
			},
			wantErr: true,
			errMsg:  "invalid new ref hash length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.req.Format()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)

			// Extract the ref line from the packet
			// The packet format is: <length><ref-line>0000<pack-file><flush>
			// Skip the length prefix (4 bytes) and extract just the ref line
			refLine := string(got[4 : len(got)-len(protocol.EmptyPack)-4]) // remove length prefix, pack file, and flush packet
			assert.Equal(t, tt.wantPkt, refLine)

			// Verify the packet structure
			assert.Equal(t, protocol.EmptyPack, got[len(got)-len(protocol.EmptyPack)-4:len(got)-4], "pack file should be present")
			assert.Equal(t, []byte("0000"), got[len(got)-4:], "should end with flush packet")
		})
	}
}

func TestNewCreateRefRequest(t *testing.T) {
	tests := []struct {
		name    string
		newRef  string
		refName string
		want    protocol.RefUpdateRequest
	}{
		{
			name:    "create main branch",
			newRef:  "1234567890123456789012345678901234567890",
			refName: "refs/heads/main",
			want: protocol.RefUpdateRequest{
				OldRef:  protocol.ZeroHash,
				NewRef:  "1234567890123456789012345678901234567890",
				RefName: "refs/heads/main",
			},
		},
		{
			name:    "create feature branch",
			newRef:  "0987654321098765432109876543210987654321",
			refName: "refs/heads/feature",
			want: protocol.RefUpdateRequest{
				OldRef:  protocol.ZeroHash,
				NewRef:  "0987654321098765432109876543210987654321",
				RefName: "refs/heads/feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newRef, err := hash.FromHex(tt.newRef)
			require.NoError(t, err)
			got := protocol.NewCreateRefRequest(tt.refName, newRef)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewUpdateRefRequest(t *testing.T) {
	tests := []struct {
		name    string
		oldRef  string
		newRef  string
		refName string
		want    protocol.RefUpdateRequest
	}{
		{
			name:    "update main branch",
			oldRef:  "1234567890123456789012345678901234567890",
			newRef:  "0987654321098765432109876543210987654321",
			refName: "refs/heads/main",
			want: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  "0987654321098765432109876543210987654321",
				RefName: "refs/heads/main",
			},
		},
		{
			name:    "update feature branch",
			oldRef:  "1111111111111111111111111111111111111111",
			newRef:  "2222222222222222222222222222222222222222",
			refName: "refs/heads/feature",
			want: protocol.RefUpdateRequest{
				OldRef:  "1111111111111111111111111111111111111111",
				NewRef:  "2222222222222222222222222222222222222222",
				RefName: "refs/heads/feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRef, err := hash.FromHex(tt.oldRef)
			require.NoError(t, err)
			newRef, err := hash.FromHex(tt.newRef)
			require.NoError(t, err)
			got := protocol.NewUpdateRefRequest(oldRef, newRef, tt.refName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewDeleteRefRequest(t *testing.T) {
	tests := []struct {
		name    string
		oldRef  string
		refName string
		want    protocol.RefUpdateRequest
	}{
		{
			name:    "delete main branch",
			oldRef:  "1234567890123456789012345678901234567890",
			refName: "refs/heads/main",
			want: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  protocol.ZeroHash,
				RefName: "refs/heads/main",
			},
		},
		{
			name:    "delete feature branch",
			oldRef:  "0987654321098765432109876543210987654321",
			refName: "refs/heads/feature",
			want: protocol.RefUpdateRequest{
				OldRef:  "0987654321098765432109876543210987654321",
				NewRef:  protocol.ZeroHash,
				RefName: "refs/heads/feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldRef, err := hash.FromHex(tt.oldRef)
			require.NoError(t, err)
			got := protocol.NewDeleteRefRequest(oldRef, tt.refName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRefLine(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantRef  string
		wantHash string
		wantErr  bool
	}{
		{
			name:     "empty line",
			input:    []byte(""),
			wantRef:  "",
			wantHash: "0000000000000000000000000000000000000000",
			wantErr:  false,
		},
		{
			name:     "flush packet",
			input:    []byte("0000"),
			wantRef:  "",
			wantHash: "0000000000000000000000000000000000000000",
			wantErr:  false,
		},
		{
			name:     "capability line",
			input:    []byte("=capability"),
			wantRef:  "",
			wantHash: "0000000000000000000000000000000000000000",
			wantErr:  false,
		},
		{
			name:     "valid ref line",
			input:    []byte("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/main"),
			wantRef:  "refs/heads/main",
			wantHash: "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
			wantErr:  false,
		},
		{
			name:     "valid ref line with capabilities",
			input:    []byte("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d refs/heads/main\000report-status-v2"),
			wantRef:  "refs/heads/main",
			wantHash: "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
			wantErr:  false,
		},
		{
			name:     "HEAD with symref",
			input:    []byte("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d HEAD symref=HEAD:refs/heads/main"),
			wantRef:  "refs/heads/main",
			wantHash: "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
			wantErr:  false,
		},
		{
			name:     "invalid hash length",
			input:    []byte("123 refs/heads/main"),
			wantRef:  "",
			wantHash: "0000000000000000000000000000000000000000",
			wantErr:  true,
		},
		{
			name:     "invalid format - missing space",
			input:    []byte("7fd1a60b01f91b314f59955a4e4d4e80d8edf11drefs/heads/main"),
			wantRef:  "",
			wantHash: "0000000000000000000000000000000000000000",
			wantErr:  true,
		},
		{
			name:     "valid ref line with extra spaces",
			input:    []byte("7fd1a60b01f91b314f59955a4e4d4e80d8edf11d  refs/heads/main  "),
			wantRef:  "refs/heads/main",
			wantHash: "7fd1a60b01f91b314f59955a4e4d4e80d8edf11d",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := protocol.ParseRefLine(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRefLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.RefName != tt.wantRef {
				t.Errorf("ParseRefLine() gotRef = %v, want %v", got.RefName, tt.wantRef)
			}
			if got.Hash.String() != tt.wantHash {
				t.Errorf("ParseRefLine() gotHash = %v, want %v", got.Hash, tt.wantHash)
			}
		})
	}
}
