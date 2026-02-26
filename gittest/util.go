package gittest

import (
	"fmt"
	"math/rand"
	"time"
)

// RandomRepoName generates a unique repository name for testing.
//
// The name includes a timestamp and random component to ensure uniqueness
// across parallel test execution. Format: "testrepo-{timestamp}-{random}"
//
// Example:
//
//	name := gittest.RandomRepoName()
//	// Returns: "testrepo-1709123456-a3f2"
func RandomRepoName() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("testrepo-%d-%04x", time.Now().Unix(), rng.Intn(0xffff))
}
