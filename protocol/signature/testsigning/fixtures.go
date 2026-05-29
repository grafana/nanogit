// Package testsigning loads the shared GPG, SSH, and S/MIME test keys.
package testsigning

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"golang.org/x/crypto/ssh"
)

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

func LoadGPG(t testing.TB) GPG {
	t.Helper()
	armoredPublic := read(t, "gpg.pub.asc")
	keyPath := fixturePath("gpg.key.asc")
	armoredKey, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read fixture gpg.key.asc: %v", err)
	}
	entities, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(armoredPublic))
	if err != nil {
		t.Fatalf("parse gpg pub: %v", err)
	}
	if len(entities) == 0 {
		t.Fatalf("gpg pub fixture contained no entities")
	}
	return GPG{
		Entity:        entities[0],
		ArmoredKey:    armoredKey,
		ArmoredPublic: armoredPublic,
		KeyPath:       keyPath,
	}
}

func LoadSSH(t testing.TB) SSH {
	t.Helper()
	priv := read(t, "ssh.ed25519")
	pubLine := read(t, "ssh.ed25519.pub")
	pub, _, _, _, err := ssh.ParseAuthorizedKey(pubLine)
	if err != nil {
		t.Fatalf("parse ssh pub: %v", err)
	}
	return SSH{PublicKey: pub, PrivateKey: priv, PublicLine: pubLine, PrivateKeyPath: fixturePath("ssh.ed25519")}
}

func LoadSMIME(t testing.TB) SMIME {
	t.Helper()
	certPath := fixturePath("smime.cert.pem")
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read fixture smime.cert.pem: %v", err)
	}
	keyPEM := read(t, "smime.key.pem")
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("smime cert fixture has no PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse smime cert: %v", err)
	}
	return SMIME{Certificate: cert, CertPEM: certPEM, KeyPEM: keyPEM, CertPath: certPath, KeyPath: fixturePath("smime.key.pem")}
}

func read(t testing.TB, name string) []byte {
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
