package testsigning

import (
	"crypto/x509"
	"encoding/pem"
)

type SMIME struct {
	Certificate *x509.Certificate
	CertPEM     []byte
	KeyPEM      []byte
	CertPath    string
	KeyPath     string
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
