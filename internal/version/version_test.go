package version

import (
	"strings"
	"testing"
)

func TestStringContainsIdentity(t *testing.T) {
	s := String()
	if !strings.Contains(s, "flowmax-ops-mcp") {
		t.Fatalf("String() = %q, want to contain flowmax-ops-mcp", s)
	}
}

func TestStringUsesInjectedValues(t *testing.T) {
	origVersion, origCommit, origBuildDate := Version, Commit, BuildDate
	defer func() { Version, Commit, BuildDate = origVersion, origCommit, origBuildDate }()

	Version = "1.2.3"
	Commit = "abc123"
	BuildDate = "2026-08-27T00:00:00Z"

	s := String()
	for _, want := range []string{"1.2.3", "abc123", "2026-08-27T00:00:00Z"} {
		if !strings.Contains(s, want) {
			t.Fatalf("String() = %q, want to contain %q", s, want)
		}
	}
}
