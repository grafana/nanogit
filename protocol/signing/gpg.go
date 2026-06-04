package signing

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
)

// NewGPGSigner signs with an unencrypted armored OpenPGP private key. The key is
// parsed once here and reused for every Sign call.
func NewGPGSigner(armoredKey []byte) (Signer, error) {
	entities, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(armoredKey))
	if err != nil {
		return nil, fmt.Errorf("read armored signing key: %w", err)
	}
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities found in signing key")
	}
	entity := entities[0]
	if entity.PrivateKey == nil {
		return nil, fmt.Errorf("signing key has no private component")
	}
	if entity.PrivateKey.Encrypted {
		return nil, fmt.Errorf("signing key is passphrase-protected")
	}
	return &gpgSigner{entity: entity}, nil
}

type gpgSigner struct {
	entity *openpgp.Entity
}

func (s *gpgSigner) Sign(data []byte) (string, error) {
	var sig strings.Builder
	if err := openpgp.ArmoredDetachSign(&sig, s.entity, bytes.NewReader(data), nil); err != nil {
		return "", fmt.Errorf("sign commit: %w", err)
	}
	return strings.TrimRight(sig.String(), "\n"), nil
}
