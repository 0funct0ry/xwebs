package build

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of xwebs, set by ldflags
	Version = "dev"
	// Commit is the git commit hash, set by ldflags
	Commit = "unknown"
	// BuildDate is the date the binary was built, set by ldflags
	BuildDate = "unknown"
)

// BuildInfo contains all the build information for the binary.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetBuildInfo returns the structured build information.
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a human-readable string representation of the build info.
func (b BuildInfo) String() string {
	return fmt.Sprintf("xwebs version %s (commit: %s, built: %s, go: %s, platform: %s)",
		b.Version, b.Commit, b.BuildDate, b.GoVersion, b.Platform)
}
