package client

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol/hash"

	"github.com/grafana/nanogit/log/mocks"
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
		logger := new(mocks.FakeLogger)
		client.logFetchRequest(logger, []byte("test"), opts)

		// Verify that the fetch request was logged and includes our option in the keyvals
		require.Equal(t, 2, logger.DebugCallCount()) // Should have "Send fetch request" and "Fetch request raw data"
		msg, _ := logger.DebugArgsForCall(0)
		require.Equal(t, "Send fetch request", msg)

		// Check that NoExtraObjects option is properly logged in keyvals
		// The first debug call should be "Send fetch request" with options in keyvals
		_, args := logger.DebugArgsForCall(0)
		require.Len(t, args, 4) // Should be: requestSize, <size>, options, <map>

		// Find the options map in the keyvals
		var optionsMap map[string]interface{}
		for i := 0; i < len(args); i += 2 {
			if key, ok := args[i].(string); ok && key == "options" {
				if options, ok := args[i+1].(map[string]interface{}); ok {
					optionsMap = options
					break
				}
			}
		}

		require.NotNil(t, optionsMap, "options should be logged")
		require.Equal(t, true, optionsMap["noExtraObjects"])
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
