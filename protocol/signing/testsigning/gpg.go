package testsigning

import (
	"bytes"

	"github.com/ProtonMail/go-crypto/openpgp"
)

type GPG struct {
	Entity        *openpgp.Entity
	ArmoredKey    []byte
	ArmoredPublic []byte
	KeyPath       string
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
