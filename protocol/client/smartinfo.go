package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/grafana/nanogit/log"
	"github.com/grafana/nanogit/protocol"
)

// SmartInfo retrieves reference and capability information from the remote Git repository
// using the Smart HTTP protocol.
//
// It sends a GET request to the $GIT_URL/info/refs endpoint with the specified service
// (e.g., "git-upload-pack" or "git-receive-pack") as a query parameter. This is required
// by the Git Smart Protocol v2 specification for repository discovery and capability
// negotiation.
//
// The response is parsed internally to validate the Git protocol format.
//
// See:
//   - https://git-scm.com/docs/http-protocol#_smart_clients
//   - https://git-scm.com/docs/protocol-v2#_http_transport
//
// Parameters:
//
//	ctx     - Context for request cancellation and deadlines.
//	service - The Git service to query ("git-upload-pack" or "git-receive-pack").
//
// Returns:
//
//	An error if the HTTP request fails, the server returns a non-2xx status code, or Git protocol errors are detected.
func (c *rawClient) SmartInfo(ctx context.Context, service string) error {
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", service)
	u.RawQuery = query.Encode()

	logger := log.FromContext(ctx)
	logger.Debug("SmartInfo", "url", u.String(), "service", service)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	c.addDefaultHeaders(req)

	// Retries on network errors, 5xx server errors, and 429 (Too Many Requests) for GET requests
	res, err := c.do(ctx, req)

	if err != nil {
		return err
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Check for structured client errors (401, 403, 404)
		if clientErr := CheckHTTPClientError(res); clientErr != nil {
			return clientErr
		}

		// Generic error for other non-2xx codes
		return fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	logger.Debug("SmartInfo response",
		"status", res.StatusCode,
		"statusText", res.Status)

	return nil
}

// ErrProtocolV1NotSupported is returned when a Git server only supports protocol v1.
var ErrProtocolV1NotSupported = errors.New("git protocol v1 is not supported; nanogit requires protocol v2")

// ProtocolVersion represents the detected Git protocol version.
type ProtocolVersion int

const (
	// ProtocolVersionUnknown indicates the protocol version could not be determined.
	ProtocolVersionUnknown ProtocolVersion = iota
	// ProtocolVersionV1 indicates Git protocol v1.
	ProtocolVersionV1
	// ProtocolVersionV2 indicates Git protocol v2.
	ProtocolVersionV2
)

// CheckProtocolVersion checks if the Git server supports protocol v2.
// It returns an error if the server only supports protocol v1.
//
// This method makes a single request to the /info/refs endpoint to detect the protocol version
// and returns ErrProtocolV1NotSupported if the server only supports v1.
//
// Protocol Detection:
//   - Protocol v2: Response contains "version 2" announcement or capability lines starting with '='
//   - Protocol v1: Response contains ref advertisements (hash + space + refname) without v2 indicators
//   - Unknown: No clear indicators (allows operations to proceed)
//
// Most modern Git servers support protocol v2 (introduced in Git 2.18, 2018).
//
// Parameters:
//
//	ctx     - Context for request cancellation and deadlines.
//	service - The Git service to query ("git-upload-pack" or "git-receive-pack").
//
// Returns:
//
//	The detected ProtocolVersion and an error if v1-only server is detected.
func (c *rawClient) CheckProtocolVersion(ctx context.Context, service string) (ProtocolVersion, error) {
	u := c.base.JoinPath("info/refs")

	query := make(url.Values)
	query.Set("service", service)
	u.RawQuery = query.Encode()

	logger := log.FromContext(ctx)
	logger.Debug("CheckProtocolVersion", "url", u.String(), "service", service)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return ProtocolVersionUnknown, err
	}

	c.addDefaultHeaders(req)

	// Retries on network errors, 5xx server errors, and 429 (Too Many Requests) for GET requests
	res, err := c.do(ctx, req)
	if err != nil {
		return ProtocolVersionUnknown, err
	}

	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("error closing response body: %w", closeErr)
		}
	}()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Check for structured client errors (401, 403, 404)
		if clientErr := CheckHTTPClientError(res); clientErr != nil {
			return ProtocolVersionUnknown, clientErr
		}

		// Generic error for other non-2xx codes
		return ProtocolVersionUnknown, fmt.Errorf("got status code %d: %s", res.StatusCode, res.Status)
	}

	// Parse the response to detect protocol version
	version := detectProtocolVersionFromReader(res.Body)

	if version == ProtocolVersionV1 {
		return version, fmt.Errorf("%w: server at %s only supports Git protocol v1. "+
			"Please upgrade your Git server to a version that supports protocol v2 (Git 2.18+)",
			ErrProtocolV1NotSupported, c.base.Host)
	}

	logger.Debug("Protocol version detected", "version", version)
	return version, nil
}

// detectProtocolVersionFromReader parses a Git Smart HTTP info/refs response
// to determine the protocol version.
func detectProtocolVersionFromReader(body io.Reader) ProtocolVersion {
	// Read all content from body first
	content, err := io.ReadAll(body)
	if err != nil {
		return ProtocolVersionUnknown
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
			// Check if there's more data after this position
			if err == io.EOF {
				// If we hit EOF, check if we've read all the content
				// If not, there might be more packets after a flush
				if reader.Len() > 0 {
					// Reset parser to continue reading
					parser = protocol.NewParser(reader)
					continue
				}
				break
			}
			// Ignore other parse errors - we're just doing detection
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
		return ProtocolVersionV2
	}
	if hasRefLine {
		return ProtocolVersionV1
	}
	return ProtocolVersionUnknown
}

// isHexHash checks if a byte slice contains a valid 40-character hexadecimal hash
func isHexHash(b []byte) bool {
	if len(b) != 40 {
		return false
	}
	for _, c := range b {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}
