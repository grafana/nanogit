package signature_test

import (
	"bytes"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"
	"github.com/grafana/nanogit/protocol/signature"
	"github.com/grafana/nanogit/protocol/signature/testsigning"
)

func TestGPGSigner_RoundTrip(t *testing.T) {
	t.Parallel()

	gpg := testsigning.LoadGPG(t)
	c := newTestCommit("msg")
	unsigned := c.Build(false)

	signer, err := signature.NewGPGSigner(gpg.ArmoredKey)
	require.NoError(t, err)
	sig, err := signer.Sign(unsigned)
	require.NoError(t, err)
	require.NotEmpty(t, sig)
	require.False(t, strings.HasSuffix(sig, "\n"), "trailing newline must be stripped")

	c.Signature = sig
	signed := c.Build(true)
	require.Contains(t, string(signed), "gpgsig -----BEGIN PGP SIGNATURE-----")

	_, err = openpgp.CheckArmoredDetachedSignature(
		openpgp.EntityList{gpg.Entity},
		bytes.NewReader(unsigned),
		strings.NewReader(sig),
		nil,
	)
	require.NoError(t, err)
}

func TestGPGSigner_Errors(t *testing.T) {
	t.Parallel()

	_, err := signature.NewGPGSigner([]byte("not a key"))
	require.Error(t, err)
}

func TestSSHSigner_RoundTrip(t *testing.T) {
	t.Parallel()

	k := testsigning.LoadSSH(t)
	c := newTestCommit("msg")
	unsigned := c.Build(false)

	signer, err := signature.NewSSHSigner(k.PrivateKey)
	require.NoError(t, err)
	sig, err := signer.Sign(unsigned)
	require.NoError(t, err)
	require.Contains(t, sig, "-----BEGIN SSH SIGNATURE-----")

	verifySSHSig(t, k.PublicKey, unsigned, sig)
}

func TestSSHSigner_Errors(t *testing.T) {
	t.Parallel()

	_, err := signature.NewSSHSigner([]byte("not a key"))
	require.Error(t, err)
}

func TestSMIMESigner_RoundTrip(t *testing.T) {
	t.Parallel()

	s := testsigning.LoadSMIME(t)
	c := newTestCommit("msg")
	unsigned := c.Build(false)

	signer, err := signature.NewSMIMESigner(s.KeyPEM, s.CertPEM)
	require.NoError(t, err)
	sig, err := signer.Sign(unsigned)
	require.NoError(t, err)
	require.Contains(t, sig, "-----BEGIN SIGNED MESSAGE-----")

	block, _ := pem.Decode([]byte(sig))
	require.NotNil(t, block)
	p7, err := pkcs7.Parse(block.Bytes)
	require.NoError(t, err)
	p7.Content = unsigned
	require.NoError(t, p7.VerifyWithChain(certPool(s.Certificate)))
}

func TestSMIMESigner_Errors(t *testing.T) {
	t.Parallel()

	_, err := signature.NewSMIMESigner([]byte("not a key"), []byte("not a cert"))
	require.Error(t, err)
}

func newTestCommit(msg string) *protocol.PackfileCommit {
	ident := &protocol.Identity{Name: "A", Email: "a@b", Timestamp: 1234567890, Timezone: "+0000"}
	return &protocol.PackfileCommit{
		Tree:      hash.Zero,
		Parent:    hash.Zero,
		Author:    ident,
		Committer: ident,
		Message:   msg,
	}
}

func verifySSHSig(t *testing.T, pub ssh.PublicKey, unsigned []byte, armored string) {
	t.Helper()
	block, _ := pem.Decode([]byte(armored))
	require.NotNil(t, block)

	p := block.Bytes
	require.True(t, bytes.HasPrefix(p, []byte("SSHSIG")))
	p = p[len("SSHSIG"):]
	_, p = readSSHUint32(t, p)
	_, p = readSSHString(t, p)
	namespace, p := readSSHString(t, p)
	_, p = readSSHString(t, p)
	hashAlgo, p := readSSHString(t, p)
	sigBlob, _ := readSSHString(t, p)
	var sig ssh.Signature
	require.NoError(t, ssh.Unmarshal(sigBlob, &sig))

	require.Equal(t, "git", string(namespace))
	require.Equal(t, "sha512", string(hashAlgo))

	require.NoError(t, pub.Verify(buildSSHSignedData(unsigned), &sig))
}

func buildSSHSignedData(unsigned []byte) []byte {
	out := &bytes.Buffer{}
	out.WriteString("SSHSIG")
	writeSSHString(out, []byte("git"))
	writeSSHString(out, nil)
	writeSSHString(out, []byte("sha512"))
	h := sha512.Sum512(unsigned)
	writeSSHString(out, h[:])
	return out.Bytes()
}

func writeSSHString(buf *bytes.Buffer, b []byte) {
	var l [4]byte
	l[0] = byte(len(b) >> 24)
	l[1] = byte(len(b) >> 16)
	l[2] = byte(len(b) >> 8)
	l[3] = byte(len(b))
	buf.Write(l[:])
	buf.Write(b)
}

func readSSHUint32(t *testing.T, b []byte) (uint32, []byte) {
	t.Helper()
	require.GreaterOrEqual(t, len(b), 4)
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), b[4:]
}

func readSSHString(t *testing.T, b []byte) ([]byte, []byte) {
	t.Helper()
	n, b := readSSHUint32(t, b)
	require.GreaterOrEqual(t, uint32(len(b)), n)
	return b[:n], b[n:]
}

func certPool(cert *x509.Certificate) *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool
}
