package protocol

import (
	"errors"
	"strings"
)

type RefName struct {
	// FullName is the entire, raw refname, including the 'refs/' prefix (unless it is HEAD).
	FullName string
	// Category is the first part of the refname after 'refs/'. E.g. 'heads'. Can be 'HEAD' for HEAD.
	// Does not include a final slash.
	Category string
	// Location is the final remainder of the refname, after the category. E.g. 'main', 'feature/test'. 'HEAD' does not mean this is HEAD; use 'FullName' to check for 'HEAD'.
	Location string
}

// HEAD is a special-case refname that always exists and is always valid. It refers to the current head of the tree.
var HEAD RefName = RefName{
	FullName: "HEAD",
	Category: "HEAD",
	Location: "HEAD",
}

// Parses the refname passed in.
// HEAD is always a valid refname. If given, the constant is returned.
// Otherwise, we require the refname to start with `ref/`, then follow the following rules:
//
//   - It can include a slash ('/') for hierarchical (directory) grouping. No slash-separated component can start with a dot ('.').
//   - It must contain one slash. This enforces a need for e.g. 'heads/' or 'tags/', but the actual name there is not necessary to consider.
//   - No consecutive dots can ('..') exist anywhere.
//   - They cannot contain: any byte < 40, DEL (177), space, caret ('^'), colon (':'), question mark ('?'), asterisk ('*'), open square bracket ('[').
//   - It cannot end with a slash or a dot ('/', '.').
//   - It cannot end with '.lock'.
//   - It cannot contain '@{'.
//   - It cannot contain a '\\'.
//
// See https://git-scm.com/docs/git-check-ref-format
func ParseRefName(in string) (RefName, error) {
	if in == "HEAD" {
		return HEAD, nil
	}

	rn := RefName{FullName: in}
	if !strings.HasPrefix(in, "refs/") {
		return rn, errors.New("ref name does not include refs/ prefix")
	}
	in = in[len("refs/"):]

	sepIdx := strings.IndexRune(in, '/')
	if sepIdx == -1 {
		return rn, errors.New("ref name does not include a category")
	}

	// The performance of this function could be improved, possibly by
	// implementin a state machine, but we need a reference point first.

	if strings.Contains(in, "..") {
		return rn, errors.New("ref cannot have two consecutive dots `..` anywhere")
	}

	if strings.Contains(in, "//") {
		return rn, errors.New("ref cannot contain multiple consecutive slashes")
	}

	if strings.Contains(in, "@{") {
		return rn, errors.New("ref cannot contain a sequence `@{`")
	}

	if strings.HasSuffix(in, ".") {
		return rn, errors.New("ref cannot end with a dot `.`")
	}

	for _, component := range strings.Split(in, "/") {
		if component == "" {
			return rn, errors.New("ref components cannot be empty")
		}

		if component == "@" {
			return rn, errors.New("ref components cannot be the single character `@`")
		}

		if strings.HasPrefix(component, ".") {
			return rn, errors.New("ref components cannot begin with a dot `.` or end with the sequence .lock")
		}

		if strings.HasSuffix(component, ".lock") {
			return rn, errors.New("ref components cannot end with the sequence `.lock`")
		}

		hasInvalidRunes := strings.ContainsFunc(component, func(r rune) bool {
			return r < 0o040 || r == 0o177 || r == ' ' || r == '~' || r == '^' || r == ':' || r == '?' || r == '*' || r == '[' || r == '\\'
		})

		if hasInvalidRunes {
			return rn, errors.New("ref components cannot contain control characters, spaces, `~`, `^`, `:`, `?`, `*`, `[`, `DEL`, or a backslash")
		}
	}

	rn.Category = in[:sepIdx]
	rn.Location = in[sepIdx+1:]

	return rn, nil
}
