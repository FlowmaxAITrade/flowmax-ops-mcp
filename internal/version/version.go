package version

import (
	"fmt"
	"runtime/debug"
)

// These are overridden at build time via -ldflags. The git tag is the single
// source of truth: GoReleaser injects Version={{ .Version }} so a tagged
// release reports exactly its tag, and un-tagged local builds stay "dev".
var (
	// Version is the semantic version (e.g. "0.1.0"). Injected via -ldflags.
	Version = "dev"
	// Commit is the full VCS revision. Injected via -ldflags.
	Commit = ""
	// BuildDate is the build timestamp (RFC3339). Injected via -ldflags.
	BuildDate = ""
)

// String returns a human-readable build identity, e.g.
//
//	flowmax-ops-mcp 0.1.0 (commit=abc1234, built=2026-08-27T10:00:00Z)
//
// For non-release builds (no ldflags), Version falls back to the module
// version and Commit to the embedded VCS revision when available.
func String() string {
	v := Version
	if v == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
	}
	commit := Commit
	if commit == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" {
					commit = s.Value
					break
				}
			}
		}
	}
	if commit == "" {
		commit = "unknown"
	}
	if BuildDate != "" {
		return fmt.Sprintf("flowmax-ops-mcp %s (commit=%s, built=%s)", v, commit, BuildDate)
	}
	return fmt.Sprintf("flowmax-ops-mcp %s (commit=%s)", v, commit)
}
