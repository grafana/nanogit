package integration_test

import (
	"testing"

	"github.com/grafana/nanogit/test/helpers"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	helpers.IntegrationTestSuite
}

// TestIntegrationTestSuite runs the integration test suite
func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
