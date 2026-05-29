package signature

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

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

func (s *sshSigner) Sign(data []byte) (string, error) {
	digest := sha512.Sum512(data)
	signedData := append([]byte("SSHSIG"), ssh.Marshal(struct {
		Namespace, Reserved, HashAlgo, Hash string
	}{"git", "", "sha512", string(digest[:])})...)

	// SSHSIG requires SHA-2 for RSA keys; ssh-rsa (SHA-1) is rejected by verifiers.
	var sig *ssh.Signature
	var err error
	if s.signer.PublicKey().Type() == ssh.KeyAlgoRSA {
		as, ok := s.signer.(ssh.AlgorithmSigner)
		if !ok {
			return "", fmt.Errorf("rsa key does not implement ssh.AlgorithmSigner")
		}
		sig, err = as.SignWithAlgorithm(rand.Reader, signedData, ssh.KeyAlgoRSASHA512)
	} else {
		sig, err = s.signer.Sign(rand.Reader, signedData)
	}
	if err != nil {
		return "", fmt.Errorf("ssh sign: %w", err)
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
	return strings.TrimRight(string(armored), "\n"), nil
}
