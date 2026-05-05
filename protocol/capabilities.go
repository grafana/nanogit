package protocol

import (
	"errors"
	"fmt"
	"strings"
)

// Capability names a single Git protocol capability as advertised in the
// receive-pack / upload-pack pkt-line following the NUL separator.
// See https://git-scm.com/docs/protocol-capabilities.
type Capability string

// Validate rejects capability tokens that would break pkt-line framing. Empty
// tokens splice adjacent capabilities together on the wire; NUL terminates
// the capability list prematurely; whitespace and other control characters
// split a single capability into two. The check is defensive — the constants
// in this package are valid by construction, but caller-supplied values
// (notably via CapAgent and the CLI's --receive-pack-capability flag) are not.
func (c Capability) Validate() error {
	s := string(c)
	if s == "" {
		return errors.New("capability must not be empty")
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		// Reject all ASCII control chars (<= 0x1F), space (0x20), and DEL
		// (0x7F). Git capabilities are ASCII; higher bytes are unusual but
		// we leave them to the server rather than guessing.
		if b <= 0x20 || b == 0x7f {
			return fmt.Errorf("capability %q contains invalid byte 0x%02x at index %d", s, b, i)
		}
	}
	return nil
}

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

// DefaultReceivePackCapabilities returns a fresh slice of the capabilities nanogit
// advertises by default on receive-pack ref update commands. A new slice is
// returned on every call so callers may freely mutate it.
func DefaultReceivePackCapabilities() []Capability {
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
// as the formatted DefaultReceivePackCapabilities. Each capability is
// validated first so a malformed token cannot slip into the wire protocol —
// the first invalid token short-circuits with an error.
func FormatCapabilities(caps []Capability) (string, error) {
	if len(caps) == 0 {
		caps = DefaultReceivePackCapabilities()
	}
	parts := make([]string, len(caps))
	for i, c := range caps {
		if err := c.Validate(); err != nil {
			return "", err
		}
		parts[i] = string(c)
	}
	return strings.Join(parts, " "), nil
}

// alwaysRetainedCapabilityKeys lists capability keys that IntersectCapabilities
// must keep from the client side regardless of whether the server advertised
// them. report-status-v2 is what nanogit's receive-pack response parser relies
// on to detect failures — dropping it because the server didn't advertise it
// would re-introduce silent push success on rejection. agent= is a pure
// client self-identifier with no server-side feature semantics; preserving it
// keeps server-side request logs accurate.
var alwaysRetainedCapabilityKeys = map[string]struct{}{
	"report-status-v2": {},
	"agent":            {},
}

// capabilityKey returns the comparison key for a capability token, which is
// the substring before the first "=". For valued capabilities like
// "agent=git/2.43" or "object-format=sha1" this is the bare name; flag
// capabilities like "report-status-v2" are returned unchanged.
func capabilityKey(c Capability) string {
	key, _, _ := strings.Cut(string(c), "=")
	return key
}

// IntersectCapabilities filters client to the capabilities also advertised by
// server, compared by key (the substring before "="), so a client-side
// "agent=nanogit" matches a server-side "agent=git/2.43". The client's value
// is always kept on a match — for valued capabilities like "agent=" and
// "object-format=" the client's value is what we will actually advertise on
// the wire.
//
// Capability order from the client slice is preserved. Capabilities listed in
// alwaysRetainedCapabilityKeys (report-status-v2, agent=) are kept in the
// result regardless of what the server advertised, and are injected with
// nanogit's defaults if the client did not include them — dropping them
// would either break nanogit's own response parser (report-status-v2) or
// strip the client identifier (agent=). The returned slice is therefore
// guaranteed to be non-empty: returning an empty slice would trigger the
// empty→defaults fallback in FormatCapabilities and silently re-introduce
// capabilities the caller meant to filter out.
//
// A nil or empty server slice is treated as "the server advertised nothing
// matchable" and yields only the always-retained entries. The returned
// slice is freshly allocated; mutating it does not affect either input.
func IntersectCapabilities(client, server []Capability) []Capability {
	serverKeys := make(map[string]struct{}, len(server))
	for _, c := range server {
		serverKeys[capabilityKey(c)] = struct{}{}
	}

	out := make([]Capability, 0, len(client)+len(alwaysRetainedCapabilityKeys))
	seenRetained := make(map[string]struct{}, len(alwaysRetainedCapabilityKeys))
	for _, c := range client {
		key := capabilityKey(c)
		if _, retain := alwaysRetainedCapabilityKeys[key]; retain {
			out = append(out, c)
			seenRetained[key] = struct{}{}
			continue
		}
		if _, ok := serverKeys[key]; ok {
			out = append(out, c)
		}
	}

	// Inject defaults for retained keys missing from the client list. This
	// upholds the function's contract: the result must always carry the
	// caps nanogit needs internally (report-status-v2 for the parser,
	// agent= for the client identifier), even if the caller stripped them.
	if _, ok := seenRetained["report-status-v2"]; !ok {
		out = append(out, CapReportStatusV2)
	}
	if _, ok := seenRetained["agent"]; !ok {
		out = append(out, CapAgent("nanogit"))
	}
	return out
}
