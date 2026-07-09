package integration_test

import (
	"errors"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/gittest"
	"github.com/grafana/nanogit/protocol"
	"github.com/grafana/nanogit/protocol/hash"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MergeBase", func() {
	var (
		client nanogit.Client
		local  *gittest.LocalRepo
	)

	revParse := func(rev string) hash.Hash {
		output, err := local.Git("rev-parse", rev)
		Expect(err).NotTo(HaveOccurred())
		h, err := hash.FromHex(output)
		Expect(err).NotTo(HaveOccurred())
		return h
	}

	commitFile := func(path, content, message string) hash.Hash {
		Expect(local.CreateFile(path, content)).To(Succeed())
		_, err := local.Git("add", ".")
		Expect(err).NotTo(HaveOccurred())
		_, err = local.Git("commit", "-m", message)
		Expect(err).NotTo(HaveOccurred())
		return revParse("HEAD")
	}

	BeforeEach(func() {
		client, _, local, _ = QuickSetup()
	})

	// This is the support-escalations #23127 scenario: a PR branch forks from
	// main, then an unrelated commit lands on main. A two-dot diff (main..branch)
	// wrongly reports the unrelated file; the three-dot diff via the merge base
	// reports only what the branch actually changed.
	Context("PR preview scenario", func() {
		var (
			forkPoint hash.Hash
			branchTip hash.Hash
			mainTip   hash.Hash
		)

		BeforeEach(func() {
			By("Recording the fork point on main")
			forkPoint = revParse("HEAD")

			By("Creating a PR branch that adds dashboard-a.json")
			_, err := local.Git("checkout", "-b", "feature-a")
			Expect(err).NotTo(HaveOccurred())
			branchTip = commitFile("dashboard-a.json", "{\"branch\":\"a\"}", "Add dashboard A")
			_, err = local.Git("push", "origin", "feature-a")
			Expect(err).NotTo(HaveOccurred())

			By("Landing an unrelated commit on main after the fork point")
			_, err = local.Git("checkout", "main")
			Expect(err).NotTo(HaveOccurred())
			mainTip = commitFile("dashboard-b.json", "{\"main\":\"b\"}", "Add dashboard B")
			_, err = local.Git("push", "origin", "main")
			Expect(err).NotTo(HaveOccurred())
		})

		It("resolves the merge base to the fork point", func() {
			base, err := client.MergeBase(ctx, mainTip, branchTip)
			Expect(err).NotTo(HaveOccurred())
			Expect(base).To(Equal(forkPoint))
		})

		It("is order-independent", func() {
			base1, err := client.MergeBase(ctx, mainTip, branchTip)
			Expect(err).NotTo(HaveOccurred())
			base2, err := client.MergeBase(ctx, branchTip, mainTip)
			Expect(err).NotTo(HaveOccurred())
			Expect(base1).To(Equal(base2))
			Expect(base1).To(Equal(forkPoint))
		})

		It("two-dot compare wrongly includes the unrelated main file (the bug)", func() {
			changes, err := client.CompareCommits(ctx, mainTip, branchTip)
			Expect(err).NotTo(HaveOccurred())

			paths := changePaths(changes)
			// dashboard-a.json is added on the branch; dashboard-b.json shows up
			// as a deletion purely because it exists on main but not the branch.
			Expect(paths).To(ContainElement("dashboard-a.json"))
			Expect(paths).To(ContainElement("dashboard-b.json"))
		})

		It("three-dot compare via merge base returns only the branch's change (the fix)", func() {
			base, err := client.MergeBase(ctx, mainTip, branchTip)
			Expect(err).NotTo(HaveOccurred())

			changes, err := client.CompareCommits(ctx, base, branchTip)
			Expect(err).NotTo(HaveOccurred())

			paths := changePaths(changes)
			Expect(paths).To(ConsistOf("dashboard-a.json"))
			Expect(changes[0].Status).To(Equal(protocol.FileStatusAdded))
		})
	})

	Context("edge cases", func() {
		It("returns the commit itself when both inputs are identical", func() {
			tip := revParse("HEAD")
			base, err := client.MergeBase(ctx, tip, tip)
			Expect(err).NotTo(HaveOccurred())
			Expect(base).To(Equal(tip))
		})

		It("returns the ancestor when one commit is an ancestor of the other", func() {
			ancestor := revParse("HEAD")
			descendant := commitFile("later.json", "{}", "Later commit")
			_, err := local.Git("push", "origin", "main")
			Expect(err).NotTo(HaveOccurred())

			base, err := client.MergeBase(ctx, ancestor, descendant)
			Expect(err).NotTo(HaveOccurred())
			Expect(base).To(Equal(ancestor))
		})

		It("errors when the commits share no common ancestor", func() {
			original := revParse("HEAD")

			By("Creating an orphan branch with unrelated history")
			_, err := local.Git("checkout", "--orphan", "orphan-branch")
			Expect(err).NotTo(HaveOccurred())
			_, err = local.Git("rm", "-rf", ".")
			Expect(err).NotTo(HaveOccurred())
			orphanTip := commitFile("orphan.json", "{}", "Orphan root")
			_, err = local.Git("push", "origin", "orphan-branch")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.MergeBase(ctx, original, orphanTip)
			Expect(errors.Is(err, nanogit.ErrNoMergeBase)).To(BeTrue())
		})
	})

	Context("merge commits", func() {
		It("preserves all parents and finds the merge base across a merge", func() {
			forkPoint := revParse("HEAD")

			By("Creating and merging a feature branch back into main")
			_, err := local.Git("checkout", "-b", "feature-merge")
			Expect(err).NotTo(HaveOccurred())
			featureTip := commitFile("feature.json", "{}", "Feature work")

			_, err = local.Git("checkout", "main")
			Expect(err).NotTo(HaveOccurred())
			mainWork := commitFile("main-work.json", "{}", "Main work")

			_, err = local.Git("merge", "--no-ff", "-m", "Merge feature-merge", "feature-merge")
			Expect(err).NotTo(HaveOccurred())
			mergeCommit := revParse("HEAD")
			_, err = local.Git("push", "origin", "main")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the merge commit exposes both parents")
			commit, err := client.GetCommit(ctx, mergeCommit)
			Expect(err).NotTo(HaveOccurred())
			Expect(commit.Parents).To(HaveLen(2))
			Expect(commit.Parents).To(ContainElements(mainWork, featureTip))
			Expect(commit.Parent).To(Equal(commit.Parents[0]))

			By("Verifying merge base of the merge commit and the fork point")
			base, err := client.MergeBase(ctx, mergeCommit, forkPoint)
			Expect(err).NotTo(HaveOccurred())
			Expect(base).To(Equal(forkPoint))
		})
	})
})

// changePaths extracts the head-side paths from a list of file changes.
func changePaths(changes []nanogit.CommitFile) []string {
	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		paths = append(paths, change.Path)
	}
	return paths
}
