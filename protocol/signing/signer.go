// Package signing provides commit signers for GPG, SSH, and S/MIME keys.
package signing

// Signer signs raw commit bytes and returns the armored signature.
type Signer interface {
	Sign(data []byte) (string, error)
}
