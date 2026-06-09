package testsigning

import (
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	PublicKey      ssh.PublicKey
	PrivateKey     []byte
	PublicLine     []byte
	PrivateKeyPath string
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
