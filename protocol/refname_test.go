package protocol_test

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"

	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
	"github.com/stretchr/testify/require"
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
