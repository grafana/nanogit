package protocol_test

import (
	"strings"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want protocol.Capability
	}{
		{name: "nanogit", in: "nanogit", want: "agent=nanogit"},
		{name: "app with version", in: "my-app/1.2.3", want: "agent=my-app/1.2.3"},
		{name: "empty string still produces agent= prefix", in: "", want: "agent="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, protocol.CapAgent(tt.in))
		})
	}
}

func TestDefaultReceivePackCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("contains the expected capabilities", func(t *testing.T) {
		got := protocol.DefaultReceivePackCapabilities()
		assert.Contains(t, got, protocol.CapReportStatusV2)
		assert.Contains(t, got, protocol.CapSideBand64k)
		assert.Contains(t, got, protocol.CapQuiet)
		assert.Contains(t, got, protocol.CapObjectFormatSHA1)
		assert.Contains(t, got, protocol.CapAgent("nanogit"))
	})

	t.Run("returns a fresh slice so callers can mutate without aliasing", func(t *testing.T) {
		a := protocol.DefaultReceivePackCapabilities()
		b := protocol.DefaultReceivePackCapabilities()
		require.Equal(t, a, b)

		// Mutating a must not affect b.
		a[0] = protocol.Capability("tampered")
		assert.NotEqual(t, a[0], b[0])
		assert.Equal(t, protocol.CapReportStatusV2, b[0])
	})
}

func TestFormatCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("joins caller-supplied capabilities with single spaces", func(t *testing.T) {
		caps := []protocol.Capability{
			protocol.CapReportStatusV2,
			protocol.CapQuiet,
			protocol.CapAgent("nanogit"),
		}
		got, err := protocol.FormatCapabilities(caps)
		require.NoError(t, err)
		assert.Equal(t, "report-status-v2 quiet agent=nanogit", got)
	})

	t.Run("preserves the order the caller provided", func(t *testing.T) {
		caps := []protocol.Capability{
			protocol.CapAgent("nanogit"),
			protocol.CapReportStatusV2,
		}
		got, err := protocol.FormatCapabilities(caps)
		require.NoError(t, err)
		assert.Equal(t, "agent=nanogit report-status-v2", got)
	})

	t.Run("nil falls back to formatted defaults", func(t *testing.T) {
		got, err := protocol.FormatCapabilities(nil)
		require.NoError(t, err)
		want, err := protocol.FormatCapabilities(protocol.DefaultReceivePackCapabilities())
		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Contains(t, got, string(protocol.CapSideBand64k))
	})

	t.Run("empty slice falls back to formatted defaults", func(t *testing.T) {
		got, err := protocol.FormatCapabilities([]protocol.Capability{})
		require.NoError(t, err)
		want, err := protocol.FormatCapabilities(protocol.DefaultReceivePackCapabilities())
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("single capability renders without trailing whitespace", func(t *testing.T) {
		got, err := protocol.FormatCapabilities([]protocol.Capability{protocol.CapQuiet})
		require.NoError(t, err)
		assert.Equal(t, "quiet", got)
		assert.False(t, strings.HasSuffix(got, " "))
	})

	t.Run("rejects invalid capability", func(t *testing.T) {
		got, err := protocol.FormatCapabilities([]protocol.Capability{protocol.CapQuiet, "bad token"})
		require.Error(t, err)
		assert.Empty(t, got)
	})
}

func TestCapability_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		c       protocol.Capability
		wantErr bool
	}{
		{name: "plain token", c: "report-status-v2", wantErr: false},
		{name: "token with equals", c: "object-format=sha1", wantErr: false},
		{name: "agent with slash and version", c: protocol.CapAgent("nanogit/1.2.3"), wantErr: false},
		{name: "empty string", c: "", wantErr: true},
		{name: "contains space", c: "report status", wantErr: true},
		{name: "agent with embedded space", c: protocol.CapAgent("nano git"), wantErr: true},
		{name: "contains tab", c: "report\tstatus", wantErr: true},
		{name: "contains NUL", c: "report\x00status", wantErr: true},
		{name: "contains newline", c: "report\nstatus", wantErr: true},
		{name: "contains carriage return", c: "report\rstatus", wantErr: true},
		{name: "contains DEL", c: "report\x7fstatus", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.c.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCapabilityConstants(t *testing.T) {
	t.Parallel()

	// Guard the wire-level string values of the exported capability constants.
	// Changing any of these is a protocol-visible change and should be intentional.
	tests := []struct {
		cap  protocol.Capability
		want string
	}{
		{protocol.CapReportStatusV2, "report-status-v2"},
		{protocol.CapSideBand64k, "side-band-64k"},
		{protocol.CapQuiet, "quiet"},
		{protocol.CapObjectFormatSHA1, "object-format=sha1"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, string(tt.cap))
	}
}
