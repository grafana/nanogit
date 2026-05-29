// Package signature provides commit signers for GPG, SSH, and S/MIME keys.
package signature

// Signer signs raw commit bytes and returns the armored signature.
type Signer interface {
	Sign(data []byte) (string, error)
}
