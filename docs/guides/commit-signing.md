# Commit signing

A `StagedWriter` can cryptographically sign every commit it creates. Three signature formats are supported, selected with a writer option at `NewStagedWriter` time:

| Option | Format | Key material |
| ------ | ------ | ------------ |
| `nanogit.WithGPGSigner(armoredKey)` | OpenPGP (`gpgsig`) | unencrypted armored private key |
| `nanogit.WithSSHSigner(privateKey)` | SSH signature | unencrypted SSH private key (OpenSSH PEM) |
| `nanogit.WithSMIMESigner(privateKey, certificate)` | S/MIME (X.509) | unencrypted PEM private key + certificate |

All keys must be **unencrypted**: decrypt passphrase-protected keys before handing them to nanogit (for example with `x/crypto/openpgp` or `x/crypto/ssh` in your key-loading code), and treat them like any other production secret.

## GPG

```go
armoredKey, err := os.ReadFile("signing-key.asc") // ASCII-armored, unencrypted
if err != nil {
    return err
}

writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithGPGSigner(armoredKey))
if err != nil {
    return err
}

// Every Commit created by this writer now carries a gpgsig header.
if _, err := writer.CreateBlob(ctx, "signed.txt", []byte("signed content\n")); err != nil {
    return err
}
if _, err := writer.Commit(ctx, "Signed commit", author, committer); err != nil {
    return err
}
if err := writer.Push(ctx); err != nil {
    return err
}
```

## SSH

```go
sshKey, err := os.ReadFile("id_ed25519") // unencrypted OpenSSH private key
if err != nil {
    return err
}
writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithSSHSigner(sshKey))
```

## S/MIME

```go
key, err := os.ReadFile("smime-key.pem")
if err != nil {
    return err
}
cert, err := os.ReadFile("smime-cert.pem")
if err != nil {
    return err
}
writer, err := client.NewStagedWriter(ctx, ref, nanogit.WithSMIMESigner(key, cert))
```

## Getting a "Verified" badge

The signature is only half the story — providers display **Verified** when the signing identity checks out on their side:

- The key (GPG/SSH) or certificate must be registered with the provider account that matches the **committer email**
- The author/committer email you pass to `Commit` must match an email on that account
- Provider support varies: GitHub verifies GPG, SSH, and S/MIME; GitLab and Gitea verify GPG and SSH. Test against your provider before relying on the badge

## Scope: signing, not verification

nanogit **signs** commits; it does not **verify** signatures on commits it reads. Signature verification is an explicit [non-goal](../index.md#when-should-i-not-use-it) — verifying is the server's (or your security tooling's) job.
