package version

import (
	"fmt"
	"runtime"
	"strings"
)

// These variables are set at build time via -ldflags.
// See Makefile and .goreleaser.yml.
var (
	// Version is the semantic version (e.g. "0.2.0").
	Version = "0.1.0-dev"

	// GitCommit is the full Git SHA of the build.
	GitCommit = "unknown"

	// GitTreeState indicates whether the working tree was clean ("clean") or
	// had uncommitted changes ("dirty") at build time.
	GitTreeState = "unknown"

	// BuildDate is the ISO-8601 UTC timestamp of the build.
	BuildDate = "unknown"
)

// Info returns a structured summary of build metadata.
type Info struct {
	Version      string `json:"version"`
	GitCommit    string `json:"git_commit"`
	GitTreeState string `json:"git_tree_state"`
	BuildDate    string `json:"build_date"`
	GoVersion    string `json:"go_version"`
	Platform     string `json:"platform"`
}

// GetInfo returns the current build info.
func GetInfo() Info {
	return Info{
		Version:      Version,
		GitCommit:    GitCommit,
		GitTreeState: GitTreeState,
		BuildDate:    BuildDate,
		GoVersion:    runtime.Version(),
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// HumanVersion returns a human-friendly version string, e.g.:
// "InfraGraph v0.2.0 (abc1234, clean, 2026-04-08T12:00:00Z)"
func HumanVersion() string {
	version := Version

	// Strip any leading "v" for display consistency.
	version = strings.TrimPrefix(version, "v")

	commit := GitCommit
	if len(commit) > 7 {
		commit = commit[:7]
	}

	return fmt.Sprintf("InfraGraph v%s (%s, %s, %s)",
		version, commit, GitTreeState, BuildDate)
}
