package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every lemmego module the scaffold imports has to be pinned.
//
// github.com/lemmego/templ was not. stubs/web.go.tpl imported it and neither
// go.mod.tpl nor versions.json mentioned it, so the `go mod tidy` that
// `lemmego new` runs resolved it to whatever was newest on the day — which
// defeats the entire point of pinning, for all three templ presets, silently.
// It was found by a compile run printing "finding module for package
// github.com/lemmego/templ", which is not where anybody looks.
//
// templateData.Version returns the literal string "latest" for an unpinned
// package (scaffold_engine.go), and "latest" is not a valid version in a
// require directive — so a missing entry for a module that *is* in go.mod.tpl
// produces a go.mod that cannot be read at all. Either way the answer is the
// same: the pin table is the source of truth and this test is what keeps it
// one.
func TestEveryScaffoldedLemmegoImportIsPinned(t *testing.T) {
	pins := scaffoldPins(t)

	// Captures the module path, which for these is always the first three
	// path segments: github.com/lemmego/<name>. A subpackage such as
	// api/providers/fs is pinned by its module, not by its own path.
	importRE := regexp.MustCompile(`"(github\.com/lemmego/[a-z0-9-]+)(?:/[^"]*)?"`)

	seen := map[string][]string{}
	err := filepath.WalkDir("_scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch filepath.Ext(path) {
		case ".go", ".tpl", ".templ":
		default:
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range importRE.FindAllStringSubmatch(string(raw), -1) {
			module := match[1]
			seen[module] = append(seen[module], path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the scaffold: %v", err)
	}
	if len(seen) == 0 {
		t.Fatal("found no lemmego imports in the scaffold, so this test proves nothing")
	}

	for module, files := range seen {
		// The example app's own templates package is rewritten to the
		// generated project's module name by renameModule, so it is never
		// resolved from the network and never needs a pin.
		if module == "github.com/lemmego/lemmego" {
			continue
		}
		if _, ok := pins[module]; !ok {
			t.Errorf("%s is imported by %s but pinned in no versions.json entry;\n"+
				"an unpinned module floats to whatever `go mod tidy` finds",
				module, strings.Join(files, ", "))
		}
	}
}

// And the reverse: a pin nobody imports is misleading rather than harmful,
// but it is still a claim the scaffold is making about itself.
//
// Reported rather than failed, because a pin can legitimately exist for a
// module reached indirectly — and because the point is to notice the drift,
// not to forbid it.
func TestPinsWithoutAnImportAreReported(t *testing.T) {
	pins := scaffoldPins(t)

	var text strings.Builder
	err := filepath.WalkDir("_scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text.Write(raw)
		return nil
	})
	if err != nil {
		t.Fatalf("walking the scaffold: %v", err)
	}

	for module := range pins {
		if !strings.HasPrefix(module, "github.com/lemmego/") {
			continue
		}
		if !strings.Contains(text.String(), module) {
			t.Logf("note: %s is pinned but appears nowhere in the scaffold", module)
		}
	}
}

func scaffoldPins(t *testing.T) map[string]string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("_scaffold", "stubs", "versions.json"))
	if err != nil {
		t.Fatalf("reading versions.json: %v", err)
	}
	var pins map[string]string
	if err := json.Unmarshal(raw, &pins); err != nil {
		t.Fatalf("parsing versions.json: %v", err)
	}
	return pins
}
