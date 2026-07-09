package integration_test

import (
	"errors"
	"fmt"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MergeBase", func() {
	var (
		client nanogit.Client
		local  *gittest.LocalRepo
	)

	commitFile := func(path, content, message string) hash.Hash {
		Expect(local.CreateFile(path, content)).To(Succeed())
		_, err := local.Git("add", path)
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", message)
		Expect(err).NotTo(HaveOccurred())
		output, err := local.Git("rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())
		h, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())
		return h
	}

	gitMergeBase := func(a, b hash.Hash) hash.Hash {
		output, err := local.Git("merge-base", a.String(), b.String())
		Expect(err).NotTo(HaveOccurred())
		h, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())
		return h
	}

	BeforeEach(func() {
		client, _, local, _ = QuickSetup()
	})

	It("should find the fork point of two diverged branches", func() {
		forkPoint := commitFile("base.txt", "base", "Base commit")

		_, err := local.Git("checkout", "-b", "feature")
		Expect(err).NotTo(HaveOccurred())
		var featureHead hash.Hash
		for i := 0; i < 3; i++ {
			featureHead = commitFile(fmt.Sprintf("feature-%d.txt", i), "feature", fmt.Sprintf("Feature commit %d", i))
		}

		_, err = local.Git("checkout", "main")
		Expect(err).NotTo(HaveOccurred())
		var mainHead hash.Hash
		for i := 0; i < 3; i++ {
			mainHead = commitFile(fmt.Sprintf("main-%d.txt", i), "main", fmt.Sprintf("Main commit %d", i))
		}

		_, err = local.Git("push", "origin", "main", "feature", "--force")
		Expect(err).NotTo(HaveOccurred())

		mergeBase, err := client.MergeBase(ctx, mainHead, featureHead)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergeBase).To(Equal(forkPoint))
		Expect(mergeBase).To(Equal(gitMergeBase(mainHead, featureHead)))
	})

	It("should return the ancestor when one commit is an ancestor of the other", func() {
		older := commitFile("first.txt", "first", "First commit")
		newer := commitFile("second.txt", "second", "Second commit")

		_, err := local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		mergeBase, err := client.MergeBase(ctx, older, newer)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergeBase).To(Equal(older))

		mergeBase, err = client.MergeBase(ctx, newer, older)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergeBase).To(Equal(older))
	})

	It("should return the commit itself when both hashes are equal", func() {
		head := commitFile("only.txt", "only", "Only commit")

		_, err := local.Git("push", "origin", "main", "--force")
		Expect(err).NotTo(HaveOccurred())

		mergeBase, err := client.MergeBase(ctx, head, head)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergeBase).To(Equal(head))
	})

	It("should walk through merge commits on both sides", func() {
		commitFile("base.txt", "base", "Base commit")

		_, err := local.Git("checkout", "-b", "feature")
		Expect(err).NotTo(HaveOccurred())
		commitFile("feature.txt", "feature", "Feature commit")

		_, err = local.Git("checkout", "main")
		Expect(err).NotTo(HaveOccurred())
		commitFile("main.txt", "main", "Main commit")

		_, err = local.Git("checkout", "-b", "topic")
		Expect(err).NotTo(HaveOccurred())
		commitFile("topic.txt", "topic", "Topic commit")

		_, err = local.Git("checkout", "main")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("merge", "--no-ff", "topic", "-m", "Merge topic")
		Expect(err).NotTo(HaveOccurred())
		output, err := local.Git("rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())
		mainHead, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())

		_, err = local.Git("checkout", "feature")
		Expect(err).NotTo(HaveOccurred())
		featureHead := commitFile("feature-2.txt", "feature", "Feature commit 2")

		_, err = local.Git("push", "origin", "main", "feature", "--force")
		Expect(err).NotTo(HaveOccurred())

		mergeBase, err := client.MergeBase(ctx, mainHead, featureHead)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergeBase).To(Equal(gitMergeBase(mainHead, featureHead)))
	})

	It("should pick the newest common ancestor when the branch merged in the base", func() {
		commitFile("base.txt", "base", "Base commit")

		_, err := local.Git("checkout", "-b", "feature")
		Expect(err).NotTo(HaveOccurred())
		commitFile("feature.txt", "feature", "Feature commit")

		_, err = local.Git("checkout", "main")
		Expect(err).NotTo(HaveOccurred())
		commitFile("main.txt", "main", "Main commit")
		output, err := local.Git("rev-parse", "HEAD")
		Expect(err).NotTo(HaveOccurred())
		mainHead, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())

		_, err = local.Git("checkout", "feature")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("merge", "--no-ff", "main", "-m", "Merge main into feature")
		Expect(err).NotTo(HaveOccurred())
		featureHead := commitFile("feature-2.txt", "feature", "Feature commit 2")

		_, err = local.Git("push", "origin", "main", "feature", "--force")
		Expect(err).NotTo(HaveOccurred())

		mergeBase, err := client.MergeBase(ctx, mainHead, featureHead)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergeBase).To(Equal(mainHead))
		Expect(mergeBase).To(Equal(gitMergeBase(mainHead, featureHead)))
	})

	It("should return ErrNoMergeBase for unrelated histories", func() {
		mainHead := commitFile("main.txt", "main", "Main commit")

		_, err := local.Git("checkout", "--orphan", "orphan")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("rm", "-rf", ".")
		Expect(err).NotTo(HaveOccurred())
		orphanHead := commitFile("orphan.txt", "orphan", "Orphan commit")

		_, err = local.Git("push", "origin", "main", "orphan", "--force")
		Expect(err).NotTo(HaveOccurred())

		_, err = client.MergeBase(ctx, mainHead, orphanHead)
		Expect(err).To(HaveOccurred())
		Expect(errors.Is(err, nanogit.ErrNoMergeBase)).To(BeTrue())
	})

	It("should converge quickly on long diverged branches", func() {
		forkPoint := commitFile("base.txt", "base", "Base commit")

		_, err := local.Git("checkout", "-b", "feature")
		Expect(err).NotTo(HaveOccurred())
		var featureHead hash.Hash
		for i := 0; i < 150; i++ {
			featureHead = commitFile(fmt.Sprintf("feature-%d.txt", i), "feature", fmt.Sprintf("Feature commit %d", i))
		}

		_, err = local.Git("checkout", "main")
		Expect(err).NotTo(HaveOccurred())
		var mainHead hash.Hash
		for i := 0; i < 150; i++ {
			mainHead = commitFile(fmt.Sprintf("main-%d.txt", i), "main", fmt.Sprintf("Main commit %d", i))
		}

		_, err = local.Git("push", "origin", "main", "feature", "--force")
		Expect(err).NotTo(HaveOccurred())

		mergeBase, err := client.MergeBase(ctx, mainHead, featureHead)
		Expect(err).NotTo(HaveOccurred())
		Expect(mergeBase).To(Equal(forkPoint))
	})
})
