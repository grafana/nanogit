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
// to determine the protocol version. It uses streaming parsing with early-exit
// to minimize memory usage and improve performance.
func detectProtocolVersionFromReader(body io.Reader) protocolVersion {
	// Limit read to 1MB to prevent memory exhaustion from malicious servers
	const maxResponseSize = 1024 * 1024 // 1MB
	limitedReader := io.LimitReader(body, maxResponseSize)

	// Buffer the first chunk for flush packet handling
	// Protocol v2 indicators typically appear in the first few packets
	const bufferSize = 4096
	buffer := make([]byte, bufferSize)
	n, err := io.ReadFull(limitedReader, buffer)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return protocolVersionUnknown
	}
	buffer = buffer[:n]

	// Create a multi-reader that combines our buffer with the rest of the stream
	combinedReader := io.MultiReader(bytes.NewReader(buffer), limitedReader)
	parser := protocol.NewParser(combinedReader)

	hasRefLine := false

	// Parse packets with early-exit for v2 detection
	for {
		line, err := parser.Next()
		if err != nil {
			// End of stream - stop parsing
			break
		}

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Check for protocol v2 indicators (early-exit optimization)
		// 1. Version announcement: "version 2"
		trimmed := bytes.TrimSpace(line)
		if bytes.Equal(trimmed, []byte("version 2")) {
			return protocolVersionV2 // Early exit for v2
		}

		// 2. Capability lines in v2 start with '='
		if line[0] == '=' {
			return protocolVersionV2 // Early exit for v2
		}

		// Check for protocol v1 ref advertisement format
		// Format: <40-char-hex-hash> <space> <refname> [NUL capabilities]
		// Example: "1234567890abcdef... refs/heads/main\000capability1 capability2"
		if len(line) > 41 && line[40] == ' ' {
			// This looks like a ref line (hash + space + refname)
			// Validate it's actually a hex hash
			if isHexHash(line[:40]) {
				hasRefLine = true
				// Don't exit early - continue scanning for v2 indicators
			}
		}
	}

	// Determine version based on indicators found
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
