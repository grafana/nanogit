//go:build tools

// Package tools is used to track tool dependencies.
// This is a common pattern in Go modules to ensure tools are included in go.mod.
package tools

import (
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
)
