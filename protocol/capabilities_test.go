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

func TestIntersectCapabilities(t *testing.T) {
	t.Parallel()

	t.Run("filters by key when both sides match", func(t *testing.T) {
		client := []protocol.Capability{
			protocol.CapReportStatusV2,
			protocol.CapSideBand64k,
			protocol.CapQuiet,
		}
		server := []protocol.Capability{
			protocol.CapReportStatusV2,
			protocol.CapSideBand64k,
			// no quiet
		}
		got := protocol.IntersectCapabilities(client, server)
		// agent= is auto-injected because the client list lacked it.
		assert.Equal(t,
			[]protocol.Capability{
				protocol.CapReportStatusV2,
				protocol.CapSideBand64k,
				protocol.CapAgent("nanogit"),
			},
			got,
		)
	})

	t.Run("preserves client order", func(t *testing.T) {
		client := []protocol.Capability{
			protocol.CapAgent("nanogit"),
			protocol.CapReportStatusV2,
			protocol.CapQuiet,
		}
		server := []protocol.Capability{
			protocol.CapQuiet,
			protocol.CapReportStatusV2,
			protocol.CapAgent("git/2.43"),
		}
		got := protocol.IntersectCapabilities(client, server)
		assert.Equal(t,
			[]protocol.Capability{
				protocol.CapAgent("nanogit"),
				protocol.CapReportStatusV2,
				protocol.CapQuiet,
			},
			got,
		)
	})

	t.Run("agent= matches by key and keeps client value", func(t *testing.T) {
		client := []protocol.Capability{protocol.CapAgent("nanogit")}
		server := []protocol.Capability{protocol.CapAgent("git/2.43")}
		got := protocol.IntersectCapabilities(client, server)
		// We keep our own agent string regardless of what the server
		// identified as. report-status-v2 is auto-injected because the
		// client list lacked it.
		assert.Equal(t,
			[]protocol.Capability{protocol.CapAgent("nanogit"), protocol.CapReportStatusV2},
			got,
		)
	})

	t.Run("retains report-status-v2 even when server omits it", func(t *testing.T) {
		client := []protocol.Capability{
			protocol.CapReportStatusV2,
			protocol.CapSideBand64k,
		}
		server := []protocol.Capability{
			protocol.CapSideBand64k,
		}
		got := protocol.IntersectCapabilities(client, server)
		assert.Contains(t, got, protocol.CapReportStatusV2,
			"report-status-v2 must always be retained — nanogit's parser depends on it")
		assert.Contains(t, got, protocol.CapSideBand64k)
	})

	t.Run("retains agent= even when server omits it", func(t *testing.T) {
		client := []protocol.Capability{protocol.CapAgent("nanogit")}
		server := []protocol.Capability{protocol.CapQuiet}
		got := protocol.IntersectCapabilities(client, server)
		// agent= is preserved from the client list; report-status-v2 is
		// auto-injected because the client list lacked it.
		assert.Equal(t,
			[]protocol.Capability{protocol.CapAgent("nanogit"), protocol.CapReportStatusV2},
			got,
		)
	})

	t.Run("drops side-band-64k when server does not advertise it", func(t *testing.T) {
		// The motivating GitLab case: a strict server omits side-band-64k.
		client := protocol.DefaultReceivePackCapabilities()
		server := []protocol.Capability{
			protocol.CapReportStatusV2,
			protocol.CapQuiet,
			protocol.CapObjectFormatSHA1,
		}
		got := protocol.IntersectCapabilities(client, server)
		for _, c := range got {
			assert.NotEqual(t, protocol.CapSideBand64k, c, "side-band-64k must be dropped when server doesn't advertise it")
		}
		assert.Contains(t, got, protocol.CapReportStatusV2)
		assert.Contains(t, got, protocol.CapAgent("nanogit"), "agent must be retained regardless")
	})

	t.Run("empty server slice yields only always-retained client caps", func(t *testing.T) {
		client := protocol.DefaultReceivePackCapabilities()
		got := protocol.IntersectCapabilities(client, nil)
		// Only report-status-v2 and agent= survive.
		assert.Equal(t,
			[]protocol.Capability{protocol.CapReportStatusV2, protocol.CapAgent("nanogit")},
			got,
		)
	})

	t.Run("empty client slice still yields the retained caps", func(t *testing.T) {
		// The function guarantees the result carries report-status-v2 and
		// agent= regardless of what either side supplied — otherwise an
		// empty result would hit FormatCapabilities's empty→defaults
		// fallback and silently advertise the full default set.
		got := protocol.IntersectCapabilities(nil, []protocol.Capability{protocol.CapQuiet})
		assert.Equal(t,
			[]protocol.Capability{protocol.CapReportStatusV2, protocol.CapAgent("nanogit")},
			got,
		)
	})

	t.Run("client that strips retained caps still gets them back", func(t *testing.T) {
		// User said "I want only quiet"; server doesn't advertise it. The
		// raw intersection would be empty, which is unsafe — the result
		// must still include report-status-v2 and agent=nanogit so the
		// receive-pack response is parseable and nanogit identifies itself.
		client := []protocol.Capability{protocol.CapQuiet}
		server := []protocol.Capability{protocol.CapSideBand64k}
		got := protocol.IntersectCapabilities(client, server)
		assert.Equal(t,
			[]protocol.Capability{protocol.CapReportStatusV2, protocol.CapAgent("nanogit")},
			got,
		)
	})

	t.Run("returned slice is independent of inputs", func(t *testing.T) {
		client := []protocol.Capability{protocol.CapReportStatusV2, protocol.CapQuiet}
		server := []protocol.Capability{protocol.CapQuiet}
		got := protocol.IntersectCapabilities(client, server)
		require.NotEmpty(t, got)
		got[0] = protocol.Capability("tampered")
		assert.Equal(t, protocol.CapReportStatusV2, client[0])
	})
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
