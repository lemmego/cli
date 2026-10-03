package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The version printed by `lemmego --version` must come from the build, never
// from a string in the source. A hardcoded one went stale at 0.1.59 and
// stayed there through v0.1.66 — seven releases all reporting the same wrong
// number, because a literal is always syntactically valid and so nothing ever
// failed.
//
// This walks the package looking for a version-shaped literal assigned to
// anything version-shaped, so the mistake cannot be reintroduced quietly.
func TestNoHardcodedVersionLiteral(t *testing.T) {
	semver := regexp.MustCompile(`(?i)(version|Version)\s*[:=]\s*"v?\d+\.\d+\.\d+`)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || name == "version_test.go" {
			continue
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(body), "\n") {
			if semver.MatchString(line) {
				t.Errorf("%s:%d hardcodes a version; stamp it with ldflags instead:\n\t%s",
					name, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// The stamp the release workflow applies wins over everything else.
func TestStampedVersionWins(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })

	version = "v9.9.9"
	if got := resolveVersion(); got != "v9.9.9" {
		t.Fatalf("resolveVersion() = %q, want the ldflags stamp", got)
	}
}

// Unstamped, the answer must still be something — and must not look like a
// release. "dev" is less informative than a number and much more useful,
// because it cannot be wrong.
func TestUnstampedVersionIsNotAReleaseNumber(t *testing.T) {
	original := version
	t.Cleanup(func() { version = original })
	version = ""

	got := resolveVersion()
	if got == "" {
		t.Fatal("resolveVersion() returned an empty string")
	}
	if !strings.HasPrefix(got, "dev") && !strings.HasPrefix(got, "v") {
		t.Fatalf("resolveVersion() = %q, want a dev marker or a module version", got)
	}
	if got == "(devel)" {
		t.Fatal("resolveVersion() leaked Go's (devel) placeholder")
	}
}

// The command must read the resolver rather than carry its own copy.
func TestRootCommandReportsTheResolvedVersion(t *testing.T) {
	if rootCmd.Version != resolveVersion() {
		t.Fatalf("rootCmd.Version = %q, resolveVersion() = %q", rootCmd.Version, resolveVersion())
	}
}
