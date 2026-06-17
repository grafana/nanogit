package integration_test

import (
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol/signing/testsigning"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Commit Signing Verification", func() {
	verifySignedCommit := func(signer nanogit.WriterOption, setup func(user *gittest.User)) {
		user, err := gitServer.CreateUser(ctx)
		Expect(err).NotTo(HaveOccurred())
		user.Token, err = gitServer.CreateToken(ctx, user.Username)
		Expect(err).NotTo(HaveOccurred())

		setup(user)

		repo, err := gitServer.CreateRepo(ctx, gittest.RandomRepoName(), user)
		Expect(err).NotTo(HaveOccurred())
		local, err := gittest.NewLocalRepo(ctx, gittest.WithRepoLogger(logger))
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { Expect(local.Cleanup()).To(Succeed()) })
		connInfo, err := local.InitWithRemote(user, repo)
		Expect(err).NotTo(HaveOccurred())

		client, err := nanogit.NewHTTPClient(connInfo.URL,
			options.WithBasicAuth(connInfo.Username, connInfo.Password))
		Expect(err).NotTo(HaveOccurred())
		ref, err := client.GetRef(ctx, "refs/heads/main")
		Expect(err).NotTo(HaveOccurred())

		writer, err := client.NewStagedWriter(ctx, ref, signer)
		Expect(err).NotTo(HaveOccurred())
		_, err = writer.CreateBlob(ctx, "sign-test.txt", []byte("hi"))
		Expect(err).NotTo(HaveOccurred())
		when := time.Now()
		ident := nanogit.Author{Name: testsigning.SignerName, Email: testsigning.SignerEmail, Time: when}
		commit, err := writer.Commit(ctx, "signed commit\n", ident,
			nanogit.Committer{Name: ident.Name, Email: ident.Email, Time: when})
		Expect(err).NotTo(HaveOccurred())
		Expect(writer.Push(ctx)).To(Succeed())

		verified, reason, err := gitServer.CommitVerification(ctx, user.Token,
			user.Username, repo.Name, commit.Hash.String())
		Expect(err).NotTo(HaveOccurred())
		Expect(verified).To(BeTrue(), "Gitea reported the commit as unverified (reason: %q)", reason)
	}

	It("reports a GPG-signed commit as verified", func() {
		gpg := testsigning.LoadGPG(GinkgoT())
		verifySignedCommit(nanogit.WithGPGSigner(gpg.ArmoredKey), func(user *gittest.User) {
			Expect(gitServer.SetUserPrimaryEmail(ctx, user, testsigning.SignerEmail)).To(Succeed())
			Expect(gitServer.UploadGPGKey(ctx, user.Token, gpg.ArmoredPublic)).To(Succeed())
		})
	})

	It("reports an SSH-signed commit as verified", func() {
		ssh := testsigning.LoadSSH(GinkgoT())
		verifySignedCommit(nanogit.WithSSHSigner(ssh.PrivateKey), func(user *gittest.User) {})
	})
})
