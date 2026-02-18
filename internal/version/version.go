package version

import (
	"fmt"
	"runtime"
)

// These variables are set via ldflags at build time.
// Example: go build -ldflags "-X github.com/sahithyandev/faa/internal/version.Version=1.0.0"
var (
	// Version is the semantic version of the binary
	Version = "0.0.0-dev"

	// Commit is the git commit hash
	Commit = "unknown"

	// Date is the build date in RFC3339 format
	Date = "unknown"
)

// Info returns a formatted string with all version information
func Info() string {
	return fmt.Sprintf("faa version %s\ncommit: %s\nbuilt: %s\nplatform: %s/%s",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
}

// Short returns just the version string
func Short() string {
	return Version
}
