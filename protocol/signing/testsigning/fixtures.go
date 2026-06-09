// Package testsigning loads the shared GPG, SSH, and S/MIME test keys.
package testsigning

import (
	"os"
	"path/filepath"
	"runtime"
)

// TestingT is the subset of testing.TB used to load fixtures. It is satisfied
// by both *testing.T and Ginkgo's GinkgoT().
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

func read(t TestingT, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(fixturePath(name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func fixturePath(name string) string {
	_, here, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(here), "testdata", name)
}
