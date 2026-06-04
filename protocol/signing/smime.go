package signing

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/smallstep/pkcs7"
)

// NewSMIMESigner signs with a PEM-encoded S/MIME (X.509) key and certificate.
// Both are parsed once here and reused for every Sign call.
func NewSMIMESigner(privateKey, certificate []byte) (Signer, error) {
	cert, err := parsePEMCertificate(certificate)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	key, err := parsePEMPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return &smimeSigner{cert: cert, key: key}, nil
}

type smimeSigner struct {
	cert *x509.Certificate
	key  any
}

func (s *smimeSigner) Sign(data []byte) (string, error) {
	sd, err := pkcs7.NewSignedData(data)
	if err != nil {
		return "", fmt.Errorf("new signed data: %w", err)
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if err := sd.AddSigner(s.cert, s.key, pkcs7.SignerInfoConfig{}); err != nil {
		return "", fmt.Errorf("add signer: %w", err)
	}
	sd.Detach()
	der, err := sd.Finish()
	if err != nil {
		return "", fmt.Errorf("finish: %w", err)
	}

	armored := pem.EncodeToMemory(&pem.Block{Type: "SIGNED MESSAGE", Bytes: der})
	return strings.TrimRight(string(armored), "\n"), nil
}

// parsePEMCertificate returns the first CERTIFICATE block, skipping comments and
// any other block types that PEM inputs commonly include.
func parsePEMCertificate(data []byte) (*x509.Certificate, error) {
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("no CERTIFICATE PEM block found")
		}
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
	}
}

// parsePEMPrivateKey returns the first private-key block, skipping comments and
// any other block types that PEM inputs commonly include.
func parsePEMPrivateKey(data []byte) (any, error) {
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("no supported PRIVATE KEY PEM block found")
		}
		if !strings.Contains(block.Type, "PRIVATE KEY") {
			continue
		}
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		return nil, fmt.Errorf("unsupported private key format")
	}
}
