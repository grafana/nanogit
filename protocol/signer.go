package protocol

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/smallstep/pkcs7"
	"golang.org/x/crypto/ssh"
)

// Signer signs a commit and sets c.Signature.
type Signer interface {
	Sign(c *PackfileCommit) error
}

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

func (s *gpgSigner) Sign(c *PackfileCommit) error {
	c.Signature = ""
	var sig strings.Builder
	if err := openpgp.ArmoredDetachSign(&sig, s.entity, bytes.NewReader(c.Build()), nil); err != nil {
		return fmt.Errorf("sign commit: %w", err)
	}
	c.Signature = strings.TrimRight(sig.String(), "\n")
	return nil
}

// NewSSHSigner signs with an unencrypted OpenSSH or PEM private key. The key is
// parsed once here and reused for every Sign call.
func NewSSHSigner(privateKey []byte) (Signer, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse ssh private key: %w", err)
	}
	return &sshSigner{signer: signer}, nil
}

type sshSigner struct {
	signer ssh.Signer
}

func (s *sshSigner) Sign(c *PackfileCommit) error {
	c.Signature = ""

	digest := sha512.Sum512(c.Build())
	signedData := append([]byte("SSHSIG"), ssh.Marshal(struct {
		Namespace, Reserved, HashAlgo, Hash string
	}{"git", "", "sha512", string(digest[:])})...)

	// SSHSIG requires SHA-2 for RSA keys; ssh-rsa (SHA-1) is rejected by verifiers.
	var sig *ssh.Signature
	var err error
	if s.signer.PublicKey().Type() == ssh.KeyAlgoRSA {
		as, ok := s.signer.(ssh.AlgorithmSigner)
		if !ok {
			return fmt.Errorf("rsa key does not implement ssh.AlgorithmSigner")
		}
		sig, err = as.SignWithAlgorithm(rand.Reader, signedData, ssh.KeyAlgoRSASHA512)
	} else {
		sig, err = s.signer.Sign(rand.Reader, signedData)
	}
	if err != nil {
		return fmt.Errorf("ssh sign: %w", err)
	}

	payload := ssh.Marshal(struct {
		Magic                                         [6]byte
		Version                                       uint32
		PubKey, Namespace, Reserved, HashAlgo, Signed string
	}{
		[6]byte{'S', 'S', 'H', 'S', 'I', 'G'}, 1,
		string(s.signer.PublicKey().Marshal()), "git", "", "sha512", string(ssh.Marshal(sig)),
	})

	armored := pem.EncodeToMemory(&pem.Block{Type: "SSH SIGNATURE", Bytes: payload})
	c.Signature = strings.TrimRight(string(armored), "\n")
	return nil
}

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

func (s *smimeSigner) Sign(c *PackfileCommit) error {
	c.Signature = ""

	sd, err := pkcs7.NewSignedData(c.Build())
	if err != nil {
		return fmt.Errorf("new signed data: %w", err)
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if err := sd.AddSigner(s.cert, s.key, pkcs7.SignerInfoConfig{}); err != nil {
		return fmt.Errorf("add signer: %w", err)
	}
	sd.Detach()
	der, err := sd.Finish()
	if err != nil {
		return fmt.Errorf("finish: %w", err)
	}

	armored := pem.EncodeToMemory(&pem.Block{Type: "SIGNED MESSAGE", Bytes: der})
	c.Signature = strings.TrimRight(string(armored), "\n")
	return nil
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
