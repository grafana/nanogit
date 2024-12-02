package protocol_test

import (
	"testing"

	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
	"github.com/stretchr/testify/assert"
)

func TestParseRefName(t *testing.T) {
	t.Parallel()

	t.Run("parse valid ref names", func(t *testing.T) {
		testcases := []struct {
			Full     string
			Category string
			Location string
		}{
			{"HEAD", "HEAD", "HEAD"},
			{"refs/heads/main", "heads", "main"},
			{"refs/heads/feature/test", "heads", "feature/test"},
			{"refs/heads/feature.lock/test", "heads", "feature.lock/test"}, // TODO(mem): my reading of the docs is that this is not valid
			{"refs/heads/feature./test", "heads", "feature./test"},         // TODO(mem): my reading of the docs is that this is not valid
		}

		for _, tc := range testcases {
			t.Run("parse: "+tc.Full, func(t *testing.T) {
				rn, err := protocol.ParseRefName(tc.Full)
				if assert.NoError(t, err, "expected parsing valid refname to succeed, but it failed") {
					assert.Equal(t, protocol.RefName{
						FullName: tc.Full,
						Category: tc.Category,
						Location: tc.Location,
					}, rn)
				}
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
			{"\n", "new line"},
			{"refs/", "only refs prefix"},
			{"refs//", "all empty"},
			{"refs//test", "empty category"},
			{"refs/../test", ".. category"},
			{"refs/heads/.bar", "no slash-separated component can begin with a dot ."},
			{"refs/heads/foo.lock", "no slash-separated component can end with the sequence .lock."},
			{"refs/heads/.lock", "no slash-separated component can end with the sequence .lock."},
			{"refs/heads/foo..bar", "references cannot have two consecutive dots .. anywhere."},
			{"refs/heads/foo\000bar", "references cannot have control characters."},
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
				_, err := protocol.ParseRefName(tc.Value)
				assert.Error(t, err, `expected parsing refname "%q" to fail, but it succeeded`, tc.Value)
			})
		}
	})
}
