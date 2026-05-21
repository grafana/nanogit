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

// NewGPGSigner signs with an unencrypted armored OpenPGP private key.
func NewGPGSigner(armoredKey []byte) Signer {
	return &gpgSigner{armoredKey: armoredKey}
}

type gpgSigner struct {
	armoredKey []byte
}

func (s *gpgSigner) Sign(c *PackfileCommit) error {
	c.Signature = ""
	entities, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(s.armoredKey))
	if err != nil {
		return fmt.Errorf("read armored signing key: %w", err)
	}
	if len(entities) == 0 {
		return fmt.Errorf("no entities found in signing key")
	}
	signer := entities[0]
	if signer.PrivateKey == nil {
		return fmt.Errorf("signing key has no private component")
	}
	if signer.PrivateKey.Encrypted {
		return fmt.Errorf("signing key is passphrase-protected")
	}
	var sig strings.Builder
	if err := openpgp.ArmoredDetachSign(&sig, signer, bytes.NewReader(c.Build()), nil); err != nil {
		return fmt.Errorf("sign commit: %w", err)
	}
	c.Signature = strings.TrimRight(sig.String(), "\n")
	return nil
}

// NewSSHSigner signs with an unencrypted OpenSSH or PEM private key.
func NewSSHSigner(privateKey []byte) Signer {
	return &sshSigner{privateKey: privateKey}
}

type sshSigner struct {
	privateKey []byte
}

func (s *sshSigner) Sign(c *PackfileCommit) error {
	c.Signature = ""
	signer, err := ssh.ParsePrivateKey(s.privateKey)
	if err != nil {
		return fmt.Errorf("parse ssh private key: %w", err)
	}

	digest := sha512.Sum512(c.Build())
	signedData := append([]byte("SSHSIG"), ssh.Marshal(struct {
		Namespace, Reserved, HashAlgo, Hash string
	}{"git", "", "sha512", string(digest[:])})...)

	// SSHSIG requires SHA-2 for RSA keys; ssh-rsa (SHA-1) is rejected by verifiers.
	var sig *ssh.Signature
	if signer.PublicKey().Type() == ssh.KeyAlgoRSA {
		as, ok := signer.(ssh.AlgorithmSigner)
		if !ok {
			return fmt.Errorf("rsa key does not implement ssh.AlgorithmSigner")
		}
		sig, err = as.SignWithAlgorithm(rand.Reader, signedData, ssh.KeyAlgoRSASHA512)
	} else {
		sig, err = signer.Sign(rand.Reader, signedData)
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
		string(signer.PublicKey().Marshal()), "git", "", "sha512", string(ssh.Marshal(sig)),
	})

	armored := pem.EncodeToMemory(&pem.Block{Type: "SSH SIGNATURE", Bytes: payload})
	c.Signature = strings.TrimRight(string(armored), "\n")
	return nil
}

// NewSMIMESigner signs with a PEM-encoded S/MIME (X.509) key and certificate.
func NewSMIMESigner(privateKey, certificate []byte) Signer {
	return &smimeSigner{privateKey: privateKey, certificate: certificate}
}

type smimeSigner struct {
	privateKey  []byte
	certificate []byte
}

func (s *smimeSigner) Sign(c *PackfileCommit) error {
	c.Signature = ""
	cert, err := parsePEMCertificate(s.certificate)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}
	key, err := parsePEMPrivateKey(s.privateKey)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	sd, err := pkcs7.NewSignedData(c.Build())
	if err != nil {
		return fmt.Errorf("new signed data: %w", err)
	}
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if err := sd.AddSigner(cert, key, pkcs7.SignerInfoConfig{}); err != nil {
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

func parsePEMCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParseCertificate(block.Bytes)
}

func parsePEMPrivateKey(data []byte) (any, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
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
