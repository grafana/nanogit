// Package testsigning loads the shared GPG, SSH, and S/MIME test keys.
package testsigning

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ProtonMail/go-crypto/openpgp"
	"golang.org/x/crypto/ssh"
)

// TestingT is the subset of testing.TB used to load fixtures. It is satisfied
// by both *testing.T and Ginkgo's GinkgoT().
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

type GPG struct {
	Entity        *openpgp.Entity
	ArmoredKey    []byte
	ArmoredPublic []byte
	KeyPath       string
}

type SSH struct {
	PublicKey      ssh.PublicKey
	PrivateKey     []byte
	PublicLine     []byte
	PrivateKeyPath string
}

type SMIME struct {
	Certificate *x509.Certificate
	CertPEM     []byte
	KeyPEM      []byte
	CertPath    string
	KeyPath     string
}

func LoadGPG(t TestingT) *GPG {
	t.Helper()
	armoredKey := read(t, "gpg.key.asc")
	entities, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(armoredKey))
	if err != nil {
		t.Fatalf("parse gpg key: %v", err)
	}
	if len(entities) == 0 {
		t.Fatalf("gpg key fixture contained no entities")
	}
	return &GPG{
		Entity:        entities[0],
		ArmoredKey:    armoredKey,
		ArmoredPublic: read(t, "gpg.pub.asc"),
		KeyPath:       fixturePath("gpg.key.asc"),
	}
}

func LoadSSH(t TestingT) *SSH {
	t.Helper()
	pubLine := read(t, "ssh.ed25519.pub")
	pub, _, _, _, err := ssh.ParseAuthorizedKey(pubLine)
	if err != nil {
		t.Fatalf("parse ssh pub: %v", err)
	}
	return &SSH{
		PublicKey:      pub,
		PrivateKey:     read(t, "ssh.ed25519"),
		PublicLine:     pubLine,
		PrivateKeyPath: fixturePath("ssh.ed25519"),
	}
}

func LoadSMIME(t TestingT) *SMIME {
	t.Helper()
	certPEM := read(t, "smime.cert.pem")
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("smime cert fixture has no PEM block")
		return nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse smime cert: %v", err)
	}
	return &SMIME{
		Certificate: cert,
		CertPEM:     certPEM,
		KeyPEM:      read(t, "smime.key.pem"),
		CertPath:    fixturePath("smime.cert.pem"),
		KeyPath:     fixturePath("smime.key.pem"),
	}
}

func read(t TestingT, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(fixturePath(name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func fixturePath(name string) string {
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "testdata", name)
}
