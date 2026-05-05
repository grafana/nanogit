package protocol_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/grafana/nanogit/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pktLine returns a wire-format pkt-line for raw payload data, with the
// 4-byte hex length prefix. Used to build realistic info/refs bodies in
// tests without depending on the higher-level FormatPacks helper (we want
// fine control over flush placement and arbitrary trailing bytes).
func pktLine(payload string) string {
	length := len(payload) + 4
	return strings.Join([]string{
		// %04x format, lowercase, fixed width 4.
		hex4(length),
		payload,
	}, "")
}

func hex4(n int) string {
	const digits = "0123456789abcdef"
	buf := []byte("0000")
	for i := 3; i >= 0; i-- {
		buf[i] = digits[n&0xF]
		n >>= 4
	}
	return string(buf)
}

func TestParseReceivePackInfoRefs(t *testing.T) {
	t.Parallel()

	t.Run("typical Gitea-style body", func(t *testing.T) {
		// "# service=git-receive-pack\n", flush, ref-with-caps, ref, flush.
		body := pktLine("# service=git-receive-pack\n") +
			"0000" +
			pktLine("aabbccddeeff00112233445566778899aabbccdd refs/heads/main\x00report-status-v2 side-band-64k quiet object-format=sha1 agent=git/2.43\n") +
			pktLine("00112233445566778899aabbccddeeff00112233 refs/heads/dev\n") +
			"0000"

		caps, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.NoError(t, err)
		assert.Equal(t, []protocol.Capability{
			protocol.CapReportStatusV2,
			protocol.CapSideBand64k,
			protocol.CapQuiet,
			protocol.CapObjectFormatSHA1,
			protocol.CapAgent("git/2.43"),
		}, caps)
	})

	t.Run("GitHub-style body with agent prefix", func(t *testing.T) {
		body := pktLine("# service=git-receive-pack\n") +
			"0000" +
			pktLine("0123456789abcdef0123456789abcdef01234567 refs/heads/main\x00report-status delete-refs side-band-64k quiet atomic ofs-delta agent=github/g16\n") +
			"0000"

		caps, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.NoError(t, err)
		// Spot-check the high-value entries; we don't enumerate every server token.
		assert.Contains(t, caps, protocol.CapSideBand64k)
		assert.Contains(t, caps, protocol.CapAgent("github/g16"))
		assert.Contains(t, caps, protocol.Capability("delete-refs"))
	})

	t.Run("empty repository advertises capabilities^{}", func(t *testing.T) {
		// Empty repos have no real refs; Git's discovery uses a synthetic
		// "0000...0000 capabilities^{}\x00<caps>" line so capability negotiation
		// still works.
		body := pktLine("# service=git-receive-pack\n") +
			"0000" +
			pktLine("0000000000000000000000000000000000000000 capabilities^{}\x00report-status-v2 side-band-64k object-format=sha1 agent=nano-server\n") +
			"0000"

		caps, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.NoError(t, err)
		assert.Equal(t, []protocol.Capability{
			protocol.CapReportStatusV2,
			protocol.CapSideBand64k,
			protocol.CapObjectFormatSHA1,
			protocol.CapAgent("nano-server"),
		}, caps)
	})

	t.Run("first ref has no NUL returns empty slice", func(t *testing.T) {
		body := pktLine("# service=git-receive-pack\n") +
			"0000" +
			pktLine("aabbccddeeff00112233445566778899aabbccdd refs/heads/main\n") +
			"0000"

		caps, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.NoError(t, err)
		assert.NotNil(t, caps, "should return empty (non-nil) slice to distinguish from parse error")
		assert.Empty(t, caps)
	})

	t.Run("first ref has NUL with empty cap list", func(t *testing.T) {
		body := pktLine("# service=git-receive-pack\n") +
			"0000" +
			pktLine("aabbccddeeff00112233445566778899aabbccdd refs/heads/main\x00\n") +
			"0000"

		caps, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.NoError(t, err)
		assert.NotNil(t, caps)
		assert.Empty(t, caps)
	})

	t.Run("malformed pkt-line length", func(t *testing.T) {
		// Length field "zzzz" is not valid hex.
		body := "zzzz# service=git-receive-pack\n"
		_, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.Error(t, err)
	})

	t.Run("missing service header", func(t *testing.T) {
		// First pkt-line is a regular ref instead of "# service=...".
		body := pktLine("aabbccddeeff00112233445566778899aabbccdd refs/heads/main\x00report-status-v2\n") +
			"0000"
		_, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected service header")
	})

	t.Run("empty body", func(t *testing.T) {
		_, err := protocol.ParseReceivePackInfoRefs(bytes.NewReader(nil))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty response")
	})

	t.Run("no ref lines after service header", func(t *testing.T) {
		body := pktLine("# service=git-receive-pack\n") + "0000"
		_, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no ref lines")
	})

	t.Run("trailing space in capability list", func(t *testing.T) {
		// Some servers append trailing whitespace; we should not produce empty
		// capability tokens that would later fail Validate().
		body := pktLine("# service=git-receive-pack\n") +
			"0000" +
			pktLine("aabbccddeeff00112233445566778899aabbccdd refs/heads/main\x00report-status-v2  quiet \n") +
			"0000"

		caps, err := protocol.ParseReceivePackInfoRefs(strings.NewReader(body))
		require.NoError(t, err)
		for _, c := range caps {
			assert.NoError(t, c.Validate(),
				"parser must not emit tokens that fail Validate; got %q", c)
		}
		assert.Contains(t, caps, protocol.CapReportStatusV2)
		assert.Contains(t, caps, protocol.CapQuiet)
	})
}
