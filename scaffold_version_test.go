package cli

import (
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"
)

// scaffoldVersionFile is the only thing that decides whether a scaffold change
// reaches anybody.
const scaffoldVersionFile = "_scaffold/VERSION"

// TestScaffoldChangesBumpTheVersion refuses a change under _scaffold that
// leaves _scaffold/VERSION alone.
//
// This is not tidiness. `lemmego new` serves the scaffold from a cache in
// ~/.cache/lemmego/scaffold, and it refreshes that cache only when VERSION on
// main differs from the cached copy. So a scaffold fix that does not bump this
// file reaches **nobody with a warm cache** — not on the next release, not
// ever. It has happened: a template fix was written, tested, tagged and
// published, and every machine that had already run `lemmego new` once kept
// generating the broken file.
//
// The failure has no symptom. The tests pass, the tag is correct, the fix is
// genuinely in the binary, and the output is stale.
func TestScaffoldChangesBumpTheVersion(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH")
	}
	if !gitHasRef(t, git, "origin/main") {
		t.Skip("no origin/main to compare against")
	}

	changed := changedUnderScaffold(t, git)
	if len(changed) == 0 {
		return
	}

	current, err := os.ReadFile(scaffoldVersionFile)
	if err != nil {
		t.Fatalf("reading %s: %v", scaffoldVersionFile, err)
	}
	base, ok := gitShow(t, git, "origin/main:"+scaffoldVersionFile)
	if !ok {
		// No baseline to compare against, so there is nothing to assert.
		t.Skipf("%s does not exist on origin/main", scaffoldVersionFile)
	}

	if strings.TrimSpace(string(current)) != strings.TrimSpace(base) {
		return
	}

	t.Errorf("%d file(s) under _scaffold differ from origin/main, but %s is "+
		"still %q.\n\n"+
		"`lemmego new` serves the scaffold from a disk cache and refreshes it "+
		"only when this version differs from the one on main, so these changes "+
		"reach nobody who has run it before. Bump %s.\n\nChanged:\n  %s",
		len(changed), scaffoldVersionFile, strings.TrimSpace(base),
		scaffoldVersionFile, strings.Join(changed, "\n  "))
}

// changedUnderScaffold lists every path under _scaffold that differs from
// origin/main, including files not yet committed and files not yet tracked.
//
// Both sources are needed. `git diff` does not report an untracked file, and a
// whole new overlay directory is entirely untracked — which is exactly the
// change most in need of a version bump, since it adds a module the cached
// scaffold cannot produce at all.
func changedUnderScaffold(t *testing.T, git string) []string {
	t.Helper()
	seen := map[string]bool{}

	for _, line := range gitLines(t, git, "diff", "--name-only", "origin/main", "--", "_scaffold") {
		if path := strings.TrimSpace(line); path != "" {
			seen[path] = true
		}
	}
	for _, line := range gitLines(t, git, "status", "--porcelain", "--untracked-files=all", "--", "_scaffold") {
		// " M path", "?? path", "R  old -> new"
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if _, _, renamed := strings.Cut(path, " -> "); renamed {
			_, path, _ = strings.Cut(path, " -> ")
		}
		if path != "" {
			seen[path] = true
		}
	}

	delete(seen, scaffoldVersionFile)

	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func gitLines(t *testing.T, git string, args ...string) []string {
	t.Helper()
	out, err := exec.Command(git, args...).Output()
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimRight(string(out), "\n"), "\n")
}

func gitHasRef(t *testing.T, git, ref string) bool {
	t.Helper()
	return exec.Command(git, "rev-parse", "--verify", "--quiet", ref).Run() == nil
}

func gitShow(t *testing.T, git, spec string) (string, bool) {
	t.Helper()
	out, err := exec.Command(git, "show", spec).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}
