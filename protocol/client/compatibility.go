package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

// protocolVersion represents the detected Git protocol version (internal use only).
type protocolVersion int

const (
	// protocolVersionUnknown indicates the protocol version could not be determined.
	protocolVersionUnknown protocolVersion = iota
	// protocolVersionV1 indicates Git protocol v1.
	protocolVersionV1
	// protocolVersionV2 indicates Git protocol v2.
	protocolVersionV2
)

// IsServerCompatible checks if the server supports Git protocol v2, which is required by nanogit.
//
// Returns true if the server supports protocol v2, false if it only supports v1.
// Returns an error if the protocol version cannot be determined or if there are connection/authentication issues.
func (c *rawClient) IsServerCompatible(ctx context.Context) (compatible bool, err error) {
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", "git-upload-pack")
	u.RawQuery = query.Encode()

	logger := log.FromContext(ctx)
	logger.Debug("Checking protocol compatibility", "url", u.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}

	c.addDefaultHeaders(req)

	// Retries on network errors, 5xx server errors, and 429 (Too Many Requests) for GET requests
	res, err := c.do(ctx, req)
	if err != nil {
		return false, err
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Check for structured client errors (401, 403, 404)
		if clientErr := CheckHTTPClientError(res); clientErr != nil {
			return false, clientErr
		}

		// Generic error for other non-2xx codes
		return false, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	// Parse the response to detect protocol version
	version := detectProtocolVersionFromReader(res.Body)

	switch version {
	case protocolVersionV2:
		logger.Debug("Protocol compatibility checked", "version", "v2", "compatible", true)
		return true, nil
	case protocolVersionV1:
		logger.Debug("Protocol compatibility checked", "version", "v1", "compatible", false)
		return false, nil
	default: // protocolVersionUnknown
		return false, fmt.Errorf("could not determine protocol version from server response")
	}
}

// detectProtocolVersionFromReader parses a Git Smart HTTP info/refs response
// to determine the protocol version.
func detectProtocolVersionFromReader(body io.Reader) protocolVersion {
	// Limit read to 1MB to prevent memory exhaustion from malicious servers
	const maxResponseSize = 1024 * 1024 // 1MB
	limitedReader := io.LimitReader(body, maxResponseSize)

	// Read all content from body first
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return protocolVersionUnknown
	}

	// Create a buffer reader that we can restart after flush packets
	reader := bytes.NewReader(content)
	parser := protocol.NewParser(reader)

	hasV2Indicator := false
	hasRefLine := false

	// Parse packets - continue past flush packets to read all refs
	for {
		line, err := parser.Next()
		if err != nil {
			// EOF can mean either end of stream or flush packet - check if more data remains
			if err == io.EOF && reader.Len() > 0 {
				// More data after flush packet - recreate parser to continue
				parser = protocol.NewParser(reader)
				continue
			}
			// End of stream or other error - stop parsing
			break
		}

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Check for protocol v2 indicators
		// 1. Version announcement: "version 2" or "version 2\n"
		if bytes.Contains(line, []byte("version 2")) {
			hasV2Indicator = true
			continue
		}

		// 2. Capability lines in v2 start with '='
		if line[0] == '=' {
			hasV2Indicator = true
			continue
		}

		// Check for protocol v1 ref advertisement format
		// Format: <40-char-hex-hash> <space> <refname> [NUL capabilities]
		// Example: "1234567890abcdef... refs/heads/main\000capability1 capability2"
		if len(line) > 41 && line[40] == ' ' {
			// This looks like a ref line (hash + space + refname)
			// Validate it's actually a hex hash
			if isHexHash(line[:40]) {
				hasRefLine = true
			}
		}
	}

	// Determine version based on indicators found
	if hasV2Indicator {
		return protocolVersionV2
	}
	if hasRefLine {
		return protocolVersionV1
	}
	return protocolVersionUnknown
}

// isHexHash checks if a byte slice contains a valid 40-character hexadecimal hash
func isHexHash(b []byte) bool {
	if len(b) != 40 {
		return false
	}
	for _, c := range b {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
