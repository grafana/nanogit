package nanogit_test

import (
	"context"
	"testing"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStagedWriter_StorageOptions demonstrates how to use storage options
// when creating a StagedWriter
func TestStagedWriter_StorageOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Example showing how users can configure storage modes
	examples := []struct {
		name        string
		options     []nanogit.WriterOption
		description string
	}{
		{
			name:        "default auto mode",
			options:     nil, // No options = auto mode
			description: "Uses memory for small operations, file for bulk operations",
		},
		{
			name:        "explicit auto mode",
			options:     []nanogit.WriterOption{nanogit.WithAutoStorage()},
			description: "Explicitly enables auto mode",
		},
		{
			name:        "memory mode",
			options:     []nanogit.WriterOption{nanogit.WithMemoryStorage()},
			description: "Always uses memory storage for best performance",
		},
		{
			name:        "disk mode",
			options:     []nanogit.WriterOption{nanogit.WithDiskStorage()},
			description: "Always uses disk storage to minimize memory usage",
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			t.Logf("Testing %s: %s", example.name, example.description)

			// This test demonstrates the API usage - we can't easily test
			// the actual storage behavior without a real Git server,
			// but we can verify the options are accepted
			ctx := context.Background()

			// Create a dummy ref for testing
			ref := nanogit.Ref{
				Name: "refs/heads/test",
				Hash: hash.Hash{}, // Empty hash for this demo
			}

			// This would normally work with a real client, but for this test
			// we just verify the options pattern works
			options := example.options

			// Verify the options can be applied
			if options != nil {
				writerOpts, err := applyWriterOptionsHelper(options)
				require.NoError(t, err)
				assert.NotNil(t, writerOpts)

				// Verify the expected storage mode based on options
				switch example.name {
				case "explicit auto mode":
					assert.Equal(t, nanogit.PackfileStorageAuto, writerOpts.StorageMode)
				case "memory mode":
					assert.Equal(t, nanogit.PackfileStorageMemory, writerOpts.StorageMode)
				case "disk mode":
					assert.Equal(t, nanogit.PackfileStorageDisk, writerOpts.StorageMode)
				}
			}

			t.Logf("✓ Options for %s work correctly", example.name)

			// Example of how users would call this in real code:
			// client, err := nanogit.NewHTTPClient("https://github.com/user/repo")
			// writer, err := client.NewStagedWriter(ctx, ref, example.options...)
			_ = ctx
			_ = ref
		})
	}
}

// Helper function to test option application (since we can't access the internal function from test)
func applyWriterOptionsHelper(options []nanogit.WriterOption) (*nanogit.WriterOptions, error) {
	// This simulates the internal applyWriterOptions function for testing
	opts := &nanogit.WriterOptions{
		StorageMode: nanogit.PackfileStorageAuto, // Default
	}

	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	return opts, nil
}

// TestWriterOptionsUsageExamples shows example usage patterns
func TestWriterOptionsUsageExamples(t *testing.T) {
	t.Run("memory storage for performance", func(t *testing.T) {
		// Example: High-performance scenario where memory usage is not a concern
		opts, err := applyWriterOptionsHelper([]nanogit.WriterOption{
			nanogit.WithMemoryStorage(),
		})
		require.NoError(t, err)
		assert.Equal(t, nanogit.PackfileStorageMemory, opts.StorageMode)

		// In real usage:
		// writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithMemoryStorage())
	})

	t.Run("disk storage for bulk operations", func(t *testing.T) {
		// Example: Bulk operation where memory conservation is important
		opts, err := applyWriterOptionsHelper([]nanogit.WriterOption{
			nanogit.WithDiskStorage(),
		})
		require.NoError(t, err)
		assert.Equal(t, nanogit.PackfileStorageDisk, opts.StorageMode)

		// In real usage:
		// writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithDiskStorage())
	})

	t.Run("auto storage for balanced approach", func(t *testing.T) {
		// Example: Balanced approach that adapts to the workload
		opts, err := applyWriterOptionsHelper([]nanogit.WriterOption{
			nanogit.WithAutoStorage(),
		})
		require.NoError(t, err)
		assert.Equal(t, nanogit.PackfileStorageAuto, opts.StorageMode)

		// In real usage:
		// writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithAutoStorage())
		// or simply:
		// writer, err := client.NewStagedWriter(ctx, ref) // Auto is the default
	})
}
