package testproviders_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping testproviders suite in short mode")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Providers Suite")
}
