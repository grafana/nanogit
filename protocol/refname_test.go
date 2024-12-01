package protocol_test

import (
	"testing"

	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
	"github.com/stretchr/testify/assert"
)

func TestParseRefName(t *testing.T) {
	t.Run("parse valid ref names", func(t *testing.T) {
		testcases := []struct {
			Full     string
			Category string
			Location string
		}{
			{"HEAD", "HEAD", "HEAD"},
			{"refs/heads/main", "heads", "main"},
			{"refs/heads/feature/test", "heads", "feature/test"},
			{"refs/heads/feature.lock/test", "heads", "feature.lock/test"},
			{"refs/heads/feature./test", "heads", "feature./test"},
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
			{"H", "single H character"},
			{"\n", "new line"},
			{"refs/", "only refs prefix"},
			{"refs//", "all empty"},
			{"refs//test", "empty category"},
			{"refs/../test", ".. category"},
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
			{"refs/heads/test.", "otherwise valid refname ending with a dot"},
			{"refs/heads/test/", "otherwise valid refname ending with a slash"},
			{"refs/heads/test.lock", "otherwise valid refname ending with a .lock"},
		}
		for _, tc := range testcases {
			t.Run("parse: "+tc.Name, func(t *testing.T) {
				_, err := protocol.ParseRefName(tc.Value)
				assert.Error(t, err, "expected parsing refname to fail, but it succeeded")
			})
		}
	})
}
