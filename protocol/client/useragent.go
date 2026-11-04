package client

import (
	"runtime/debug"
	"sync"
)

const modulePath = "github.com/grafana/nanogit"

// defaultUserAgent is the User-Agent sent when options.WithUserAgent is not
// set, e.g. "nanogit/v1.7.4", resolved once from the binary's build info.
var defaultUserAgent = sync.OnceValue(func() string {
	bi, _ := debug.ReadBuildInfo()
	return userAgentFromBuildInfo(bi)
})

// userAgentFromBuildInfo returns "nanogit/<version>" for the nanogit module
// version recorded in bi. When no released version is known (no build info,
// a go.work workspace, or a replace to a local path, which report "(devel)"),
// it returns the bare product token "nanogit" rather than inventing a version.
func userAgentFromBuildInfo(bi *debug.BuildInfo) string {
	if bi == nil {
		return "nanogit"
	}

	var version string
	if bi.Main.Path == modulePath {
		version = bi.Main.Version
	} else {
		for _, dep := range bi.Deps {
			if dep.Path != modulePath {
				continue
			}
			// A replaced module's own version describes the code actually
			// compiled in; the required version may not.
			if dep.Replace != nil {
				dep = dep.Replace
			}
			version = dep.Version
			break
		}
	}

	if version == "" || version == "(devel)" {
		return "nanogit"
	}
	return "nanogit/" + version
}
