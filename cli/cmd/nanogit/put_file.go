package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/grafana/nanogit"
	"github.com/grafana/nanogit/options"
	"github.com/grafana/nanogit/protocol"
	"github.com/spf13/cobra"
)

var (
	putFileMessage         string
	putFileFromFile        string
	putFileAuthor          string
	putFileCommitter       string
	putFileReceivePackCaps []string
)

func init() {
	rootCmd.AddCommand(putFileCmd)

	putFileCmd.Flags().StringVarP(&putFileMessage, "message", "m", "", "Commit message (required)")
	putFileCmd.Flags().StringVar(&putFileFromFile, "from-file", "", "Read content from a local file instead of stdin")
	putFileCmd.Flags().StringVar(&putFileAuthor, "author", "", "Author of the commit in \"Name <email>\" form (falls back to NANOGIT_AUTHOR_NAME/EMAIL)")
	putFileCmd.Flags().StringVar(&putFileCommitter, "committer", "", "Committer of the commit in \"Name <email>\" form (falls back to NANOGIT_COMMITTER_NAME/EMAIL, then author)")
	putFileCmd.Flags().StringSliceVar(&putFileReceivePackCaps, "receive-pack-capability", nil,
		"Advertised receive-pack capability (repeatable). When set, replaces the default set entirely. "+
			"Common values include report-status-v2, side-band-64k, quiet, object-format=sha1, and agent=<name>. "+
			"Arbitrary capability tokens are also accepted as an escape hatch for advanced use. "+
			"Example: --receive-pack-capability=report-status-v2 --receive-pack-capability=quiet "+
			"--receive-pack-capability=object-format=sha1 --receive-pack-capability=agent=nanogit "+
			"(drops side-band-64k to work around servers that wrap report-status in side-band channel 1).")
}

var putFileCmd = &cobra.Command{
	Use:   "put-file [<repository>] <ref> <path> [-]",
	Short: "Create or update a file on a branch in a single commit",
	Long: `Create or update a file on a branch by staging a blob, committing, and
pushing in a single step. The ref must be a branch: the change is written on
top of the branch's current tip.

Content can be provided on stdin (default) or read from a local file with
--from-file. The trailing "-" positional argument is accepted but optional.

The repository argument is optional when NANOGIT_REPO is set.

Examples:
  # Pipe content on stdin
  echo "hello" | nanogit put-file https://github.com/user/repo.git main docs/note.md -m "add note" --author "Jane <jane@example.com>"

  # Use NANOGIT_REPO env var instead of repeating the URL
  export NANOGIT_REPO=https://github.com/user/repo.git
  echo "hello" | nanogit put-file main docs/note.md -m "add note" --author "Jane <jane@example.com>"

  # Read content from a local file
  nanogit put-file https://github.com/user/repo.git main docs/note.md \
    --from-file ./local.md -m "add note" --author "Jane <jane@example.com>"

  # Author/committer via env
  NANOGIT_AUTHOR_NAME=Jane NANOGIT_AUTHOR_EMAIL=jane@example.com \
    nanogit put-file https://github.com/user/repo.git main docs/note.md -m "add note" < local.md

  # Verbose output and full wire trace
  nanogit -v put-file ...                  # Info-level
  NANOGIT_TRACE=1 nanogit -v put-file ...  # Debug-level`,
	Args: putFileArgs,
	RunE: runPutFile,
}

// putFileArgs validates positional arguments. The optional trailing "-" marks
// stdin; after peeling it off we need exactly (repo, ref, path) — or just
// (ref, path) when NANOGIT_REPO provides the repo URL.
func putFileArgs(_ *cobra.Command, args []string) error {
	envSet := os.Getenv(repoEnv) != ""

	// A trailing "-" means "read from stdin"; strip it so we can validate the
	// remaining positional count. "-" in any other position is rejected.
	if len(args) > 0 && args[len(args)-1] == "-" {
		args = args[:len(args)-1]
	}
	if slices.Contains(args, "-") {
		return fmt.Errorf(`"-" marker is only valid as the last positional argument`)
	}

	if len(args) == 3 {
		return nil
	}
	if envSet && len(args) == 2 {
		return nil
	}
	if envSet {
		return fmt.Errorf("accepts 2-3 positional args (plus optional trailing \"-\" for stdin), received %d", len(args))
	}
	return fmt.Errorf("accepts 3 positional args (plus optional trailing \"-\" for stdin), received %d (or set %s and pass 2)", len(args), repoEnv)
}

// resolvePutFileArgs extracts the repo URL, ref, path, and whether the stdin
// marker was present. Callers rely on putFileArgs having validated counts.
func resolvePutFileArgs(args []string) (repoURL, refName, filePath string, stdinMarker bool) {
	if len(args) > 0 && args[len(args)-1] == "-" {
		stdinMarker = true
		args = args[:len(args)-1]
	}
	if len(args) == 3 {
		return args[0], args[1], args[2], stdinMarker
	}
	return os.Getenv(repoEnv), args[0], args[1], stdinMarker
}

func runPutFile(cmd *cobra.Command, args []string) error {
	repoURL, refName, filePath, stdinMarker := resolvePutFileArgs(args)

	if putFileMessage == "" {
		return errors.New("--message/-m is required")
	}

	if putFileFromFile != "" && stdinMarker {
		return errors.New("cannot combine --from-file with stdin marker \"-\"")
	}

	author, err := resolveAuthor(putFileAuthor)
	if err != nil {
		return fmt.Errorf("author: %w", err)
	}
	committer, err := resolveCommitter(putFileCommitter, author)
	if err != nil {
		return fmt.Errorf("committer: %w", err)
	}

	content, err := readPutFileContent(putFileFromFile)
	if err != nil {
		return err
	}

	ctx := context.Background()
	extraOpts, err := buildPutFileClientOptions(putFileReceivePackCaps)
	if err != nil {
		return err
	}
	ctx, client, err := setupClient(ctx, repoURL, extraOpts...)
	if err != nil {
		return err
	}

	ref, err := resolveBranchRef(ctx, client, refName)
	if err != nil {
		return err
	}

	writer, err := client.NewStagedWriter(ctx, ref)
	if err != nil {
		return fmt.Errorf("create staged writer: %w", err)
	}
	defer func() { _ = writer.Cleanup(ctx) }()

	commit, err := stageAndCommit(ctx, writer, filePath, content, putFileMessage, author, committer)
	if err != nil {
		return err
	}

	if err := writer.Push(ctx); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	return outputPutFileResult(commit, filePath)
}

func readPutFileContent(fromFile string) ([]byte, error) {
	if fromFile != "" {
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return nil, fmt.Errorf("read --from-file %q: %w", fromFile, err)
		}
		return data, nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	return data, nil
}

// resolveAuthor parses the --author flag, falling back to NANOGIT_AUTHOR_NAME
// and NANOGIT_AUTHOR_EMAIL environment variables. An error is returned when
// nothing usable is available — never silently fabricate an identity.
func resolveAuthor(flagValue string) (nanogit.Author, error) {
	name, email, err := resolveIdentity(flagValue, "NANOGIT_AUTHOR_NAME", "NANOGIT_AUTHOR_EMAIL")
	if err != nil {
		return nanogit.Author{}, err
	}
	return nanogit.Author{Name: name, Email: email, Time: time.Now().UTC()}, nil
}

// resolveCommitter parses the --committer flag. When unset it falls back to
// NANOGIT_COMMITTER_NAME/EMAIL, and finally to the provided author identity.
func resolveCommitter(flagValue string, author nanogit.Author) (nanogit.Committer, error) {
	if flagValue == "" && os.Getenv("NANOGIT_COMMITTER_NAME") == "" && os.Getenv("NANOGIT_COMMITTER_EMAIL") == "" {
		return nanogit.Committer{Name: author.Name, Email: author.Email, Time: time.Now().UTC()}, nil
	}
	name, email, err := resolveIdentity(flagValue, "NANOGIT_COMMITTER_NAME", "NANOGIT_COMMITTER_EMAIL")
	if err != nil {
		return nanogit.Committer{}, err
	}
	return nanogit.Committer{Name: name, Email: email, Time: time.Now().UTC()}, nil
}

func resolveIdentity(flagValue, nameEnv, emailEnv string) (string, string, error) {
	if flagValue != "" {
		return parseIdentity(flagValue)
	}
	name := os.Getenv(nameEnv)
	email := os.Getenv(emailEnv)
	if name == "" || email == "" {
		return "", "", fmt.Errorf("identity not set: pass the flag or set both %s and %s", nameEnv, emailEnv)
	}
	return name, email, nil
}

// parseIdentity reads the git-style "Name <email>" form. The input must
// contain exactly one pair of angle brackets and end at the closing bracket
// (after trimming outer whitespace); trailing junk is rejected.
func parseIdentity(s string) (string, string, error) {
	s = strings.TrimSpace(s)
	open := strings.Index(s, "<")
	closeIdx := strings.Index(s, ">")
	if open < 0 || closeIdx < 0 || closeIdx < open {
		return "", "", fmt.Errorf("expected \"Name <email>\", got %q", s)
	}
	if strings.Count(s, "<") != 1 || strings.Count(s, ">") != 1 {
		return "", "", fmt.Errorf("expected exactly one <email> pair, got %q", s)
	}
	if closeIdx != len(s)-1 {
		return "", "", fmt.Errorf("unexpected characters after '>' in %q", s)
	}
	name := strings.TrimSpace(s[:open])
	email := strings.TrimSpace(s[open+1 : closeIdx])
	if name == "" || email == "" {
		return "", "", fmt.Errorf("expected \"Name <email>\", got %q", s)
	}
	return name, email, nil
}

// resolveBranchRef returns the full Ref for a branch name. Tags and commit
// hashes are rejected because staged writes target branch tips.
func resolveBranchRef(ctx context.Context, client nanogit.Client, ref string) (nanogit.Ref, error) {
	candidates := []string{ref}
	if !strings.HasPrefix(ref, "refs/") {
		candidates = []string{"refs/heads/" + ref, ref}
	}
	var lastErr error
	for _, candidate := range candidates {
		got, err := client.GetRef(ctx, candidate)
		if err == nil {
			if !strings.HasPrefix(got.Name, "refs/heads/") {
				return nanogit.Ref{}, fmt.Errorf("ref %q is not a branch; put-file only writes to branches", got.Name)
			}
			return got, nil
		}
		lastErr = err
	}
	return nanogit.Ref{}, fmt.Errorf("resolve branch %q: %w", ref, lastErr)
}

func stageAndCommit(ctx context.Context, writer nanogit.StagedWriter, path string, content []byte, message string, author nanogit.Author, committer nanogit.Committer) (*nanogit.Commit, error) {
	exists, err := writer.BlobExists(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("check blob existence at %q: %w", path, err)
	}
	if exists {
		if _, err := writer.UpdateBlob(ctx, path, content); err != nil {
			return nil, fmt.Errorf("update blob at %q: %w", path, err)
		}
	} else {
		if _, err := writer.CreateBlob(ctx, path, content); err != nil {
			return nil, fmt.Errorf("create blob at %q: %w", path, err)
		}
	}

	commit, err := writer.Commit(ctx, message, author, committer)
	if err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return commit, nil
}

type putFileJSON struct {
	Commit string `json:"commit"`
	Path   string `json:"path"`
}

// buildPutFileClientOptions converts the --receive-pack-capability flag into
// a nanogit options slice. When no capabilities are supplied the slice is
// empty and the caller keeps the library defaults. Whitespace around each
// value is trimmed and empty entries are rejected so "a,,b" does not silently
// add a blank capability.
func buildPutFileClientOptions(rawCaps []string) ([]options.Option, error) {
	if len(rawCaps) == 0 {
		return nil, nil
	}
	caps := make([]protocol.Capability, 0, len(rawCaps))
	for _, raw := range rawCaps {
		v := strings.TrimSpace(raw)
		if v == "" {
			return nil, errors.New("--receive-pack-capability value cannot be empty")
		}
		caps = append(caps, protocol.Capability(v))
	}
	return []options.Option{options.WithReceivePackCapabilities(caps...)}, nil
}

func outputPutFileResult(commit *nanogit.Commit, path string) error {
	if globalJSON {
		out := putFileJSON{
			Commit: commit.Hash.String(),
			Path:   path,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(out)
	}
	fmt.Println(commit.Hash.String())
	return nil
}
