package protocol

import "strings"

// Capability names a single Git protocol capability as advertised in the
// receive-pack / upload-pack pkt-line following the NUL separator.
// See https://git-scm.com/docs/protocol-capabilities.
type Capability string

// Well-known capabilities advertised by nanogit on receive-pack.
const (
	// CapReportStatusV2 asks the server to send a report-status-v2 packet
	// describing the outcome of the push.
	CapReportStatusV2 Capability = "report-status-v2"

	// CapSideBand64k lets the server multiplex progress and report-status on
	// side-band channels (1 = data, 2 = progress, 3 = error). Some servers
	// (notably certain GitLab configurations) wrap report-status in channel 1.
	CapSideBand64k Capability = "side-band-64k"

	// CapQuiet asks the server to suppress non-error progress output.
	CapQuiet Capability = "quiet"

	// CapObjectFormatSHA1 declares SHA-1 as the hash algorithm for objects.
	CapObjectFormatSHA1 Capability = "object-format=sha1"
)

// CapAgent returns the "agent=<name>" capability identifying the client.
func CapAgent(name string) Capability {
	return Capability("agent=" + name)
}

// DefaultPushCapabilities returns a fresh slice of the capabilities nanogit
// advertises by default on receive-pack ref update commands. A new slice is
// returned on every call so callers may freely mutate it.
func DefaultPushCapabilities() []Capability {
	return []Capability{
		CapReportStatusV2,
		CapSideBand64k,
		CapQuiet,
		CapObjectFormatSHA1,
		CapAgent("nanogit"),
	}
}

// FormatCapabilities renders caps as the space-separated string that follows
// the NUL byte in a ref update pkt-line. An empty or nil slice is rendered
// as the formatted DefaultPushCapabilities.
func FormatCapabilities(caps []Capability) string {
	if len(caps) == 0 {
		caps = DefaultPushCapabilities()
	}
	parts := make([]string, len(caps))
	for i, c := range caps {
		parts[i] = string(c)
	}
	return strings.Join(parts, " ")
}
