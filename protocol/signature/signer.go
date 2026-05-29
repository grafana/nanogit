// Package signature provides commit signers for GPG, SSH, and S/MIME keys.
package signature

var (
	_ Signer = (*gpgSigner)(nil)
	_ Signer = (*sshSigner)(nil)
	_ Signer = (*smimeSigner)(nil)
)

// Signer signs raw commit bytes and returns the armored signature.
type Signer interface {
	Sign(data []byte) (string, error)
}
