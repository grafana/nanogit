package metrics

// Operation identifies which Git protocol operation triggered a Recorder
// sample. It is a plain string so Recorder implementations can match, log, or
// label on it without importing this package's constants, but the values
// below are the exhaustive set nanogit reports.
type Operation = string

// Values reported as the operation argument to Recorder.HTTPRequest.
const (
	// OperationSmartInfo is the info/refs handshake used for repository
	// discovery, capability negotiation, and read/write permission checks.
	OperationSmartInfo Operation = "smart-info"
	// OperationUploadPack is the git-upload-pack endpoint used for
	// ls-refs and fetch (object retrieval).
	OperationUploadPack Operation = "upload-pack"
	// OperationReceivePack is the git-receive-pack endpoint used to push
	// objects and update refs.
	OperationReceivePack Operation = "receive-pack"
	// OperationReceivePackCapabilities is the info/refs request used to
	// discover the server's git-receive-pack capabilities before a push.
	OperationReceivePackCapabilities Operation = "receive-pack-capabilities"
	// OperationCompatibility is the info/refs request used to detect
	// whether the server supports Git protocol v2.
	OperationCompatibility Operation = "compatibility"
)
