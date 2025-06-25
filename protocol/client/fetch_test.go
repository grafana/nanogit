package client

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol/hash"
)

func TestFetchStopWhenFound(t *testing.T) {
	t.Run("processes early termination option correctly", func(t *testing.T) {
		// Create fetch options with StopWhenFound enabled
		opts := FetchOptions{
			Want: []hash.Hash{
				{0x01}, // We only want hash1
			},
			NoExtraObjects: true,
		}

		// Test that the option is properly set
		require.True(t, opts.NoExtraObjects)
		require.Len(t, opts.Want, 1)

		// Test logging includes the new option
		client := &rawClient{}
		logger := &testLogger{}
		client.logFetchRequest(logger, []byte("test"), opts)

		// Verify that the fetch request was logged and includes our option in the keyvals
		require.Len(t, logger.debugMessages, 2) // Should have "Send fetch request" and "Fetch request raw data"
		require.Equal(t, "Send fetch request", logger.debugMessages[0])

		// Check that StopWhenFound option is properly logged in keyvals
		found := false
		for i := 0; i < len(logger.keyvals); i += 2 {
			if i+1 < len(logger.keyvals) {
				if key, ok := logger.keyvals[i].(string); ok && key == "options" {
					if options, ok := logger.keyvals[i+1].(map[string]interface{}); ok {
						if noExtraObjects, exists := options["noExtraObjects"]; exists {
							require.Equal(t, true, noExtraObjects)
							found = true
							break
						}
					}
				}
			}
		}
		require.True(t, found, "noExtraObjects option should be logged")
	})

	t.Run("builds wanted hashes map correctly", func(t *testing.T) {
		// Test the logic for building wanted hashes map
		opts := FetchOptions{
			Want: []hash.Hash{
				{0x01, 0x02, 0x03},
				{0x04, 0x05, 0x06},
			},
			NoExtraObjects: true,
		}

		// Build wanted hashes map (same logic as in processPackfileResponse)
		var wantedHashes map[string]bool
		if opts.NoExtraObjects && len(opts.Want) > 0 {
			wantedHashes = make(map[string]bool, len(opts.Want))
			for _, h := range opts.Want {
				wantedHashes[h.String()] = true
			}
		}

		// Verify the map was built correctly
		require.NotNil(t, wantedHashes)
		require.Len(t, wantedHashes, 2)
		require.True(t, wantedHashes[opts.Want[0].String()])
		require.True(t, wantedHashes[opts.Want[1].String()])

		// Test that a non-wanted hash is not in the map
		unwantedHash := hash.Hash{0x07, 0x08, 0x09}
		require.False(t, wantedHashes[unwantedHash.String()])
	})

	t.Run("does not build wanted hashes map when NoExtraObjects is false", func(t *testing.T) {
		opts := FetchOptions{
			Want: []hash.Hash{
				{0x01, 0x02, 0x03},
			},
			NoExtraObjects: false,
		}

		// Build wanted hashes map (same logic as in processPackfileResponse)
		var wantedHashes map[string]bool
		if opts.NoExtraObjects && len(opts.Want) > 0 {
			wantedHashes = make(map[string]bool, len(opts.Want))
			for _, h := range opts.Want {
				wantedHashes[h.String()] = true
			}
		}

		// Should not build the map when NoExtraObjects is false
		require.Nil(t, wantedHashes)
	})
}

// testLogger is a simple test implementation of log.Logger
type testLogger struct {
	debugMessages []string
	keyvals       []interface{}
}

func (l *testLogger) Debug(msg string, keyvals ...interface{}) {
	l.debugMessages = append(l.debugMessages, msg)
	l.keyvals = append(l.keyvals, keyvals...)
}

func (l *testLogger) Info(msg string, keyvals ...interface{})  {}
func (l *testLogger) Warn(msg string, keyvals ...interface{})  {}
func (l *testLogger) Error(msg string, keyvals ...interface{}) {}
