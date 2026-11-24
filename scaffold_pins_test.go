package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/mod/semver"
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

	// versions.json is excluded, and that exclusion is the whole test.
	//
	// It lists every pinned module by path, so a walk that includes it makes
	// strings.Contains true for every pin no matter what -- the check was a
	// tautology and reported nothing, ever. Combined with a t.Logf that only
	// `-v` would have shown, it looked like a guard and was not one.
	//
	// What it failed to catch: tasker was pinned, no template emitted it and
	// nothing imported it, so the pin was decorative. The version reached a
	// generated project through queue's own go.mod instead, which is how a
	// libsql project kept booting into a tasker that predated its fix while
	// the pin table claimed otherwise.
	var text strings.Builder
	err := filepath.WalkDir("_scaffold", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if filepath.Base(path) == "versions.json" {
			return nil
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
			t.Errorf("%s is pinned but appears nowhere in the scaffold, so the "+
				"pin does nothing: no template emits it and nothing imports it", module)
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

// moduleProxy is asked what "published" means, and that is the whole point.
//
// A tag in the monorepo working tree is not published — Phase 7's lesson, that
// a fix is invisible to a consumer until it is tagged *and* pushed — and the
// failure this test exists to catch is a pin behind what `go mod tidy` can
// actually resolve for somebody else's machine.
const moduleProxy = "https://proxy.golang.org"

// deliberateHolds records a pin kept behind its latest release on purpose,
// keyed by module path, valued by the reason.
//
// An entry turns a failure into a logged note, which is the difference between
// a decision and drift. None of the six lemmego pins that had fallen behind
// was held back for a reason: api simply sat at v0.9.0 against v0.12.0, and
// every project `lemmego new` generated had a password minimum that did
// nothing.
//
// The one entry is a third-party pin, and it is the case this mechanism was
// written for.
var deliberateHolds = map[string]string{
	// TypeScript 7 is current and the React presets typecheck cleanly on it.
	// vue-tsc cannot run against it: 3.3.14 is the newest published version,
	// its peer range says typescript >=5.0.0, and it still resolves an
	// internal path inside the typescript package that 7 no longer exports —
	// `npm run typecheck` in a generated Vue project dies with
	// ERR_PACKAGE_PATH_NOT_EXPORTED from resolveTscPath before it reads a
	// single file. The build is unaffected, since Vite strips types and never
	// typechecks, which is exactly how this would have shipped unnoticed.
	//
	// versions.json is one table for every preset, so the Vue presets decide
	// this pin. Revisit when a vue-tsc release claims TypeScript 7.
	"typescript": "vue-tsc 3.3.14, the newest release, cannot resolve TypeScript 7's exports",
}

// TestEveryPinIsTheLatestPublishedVersion is the guard the two tests above are
// missing.
//
// They check that a pin *exists* and that it is not an orphan. Neither looks at
// the version, so six of them drifted — api by three minors, tasker by two —
// and nothing said so. The cost was not theoretical: the scaffolded auth input
// calls Min(8) on a string password, the fix for that shipped in api v0.12.0,
// and the pin was v0.9.0, so every project generated in between enforced no
// minimum at all.
//
// It needs the network, so an offline run skips rather than fails: breaking
// `go test ./...` on a plane would get the test deleted, and CI is where this
// has to hold. The failure is therefore CI's to report.
func TestEveryPinIsTheLatestPublishedVersion(t *testing.T) {
	if os.Getenv("GOPROXY") == "off" {
		t.Skip("GOPROXY=off, so there is nothing to ask for the latest version")
	}

	pins := scaffoldPins(t)

	var modules []string
	for module := range pins {
		if strings.HasPrefix(module, "github.com/lemmego/") {
			modules = append(modules, module)
		}
	}
	sort.Strings(modules)
	if len(modules) == 0 {
		t.Fatal("versions.json pins no lemmego module, so this test proves nothing")
	}

	// Every lookup at once: twenty-odd round trips to the same host, and
	// sequentially they cost half a minute, which is the sort of number that
	// gets a test moved behind -short and then forgotten. The answers are
	// collected before anything is judged, so the decision to skip is taken
	// once, for the whole test, rather than per module.
	latest, reachable, err := latestPublishedVersions(modules)
	if !reachable {
		// Not a judgement about any pin. Skipping the whole test rather than
		// the unreachable modules is deliberate: a partial answer reported as
		// green is how a vacuous pass happens.
		t.Skipf("cannot reach %s (%v).\nThis test needs the network; CI runs it.",
			moduleProxy, err)
	}

	for _, module := range modules {
		if lookupErr := latest[module].err; lookupErr != nil {
			t.Errorf("%s is pinned at %s but asking the proxy for its latest version "+
				"failed: %v\nA pin that cannot be resolved is a project that cannot "+
				"be generated.", module, pins[module], lookupErr)
			continue
		}

		pin := pins[module]
		newest := latest[module].version
		if reason, held := deliberateHolds[module]; held {
			if semver.Compare(pin, newest) >= 0 {
				t.Errorf("%s is pinned at %s, which is now the latest published version, "+
					"so its deliberateHolds entry is obsolete and should be removed.\n"+
					"It said: %s", module, pin, reason)
				continue
			}
			t.Logf("note: %s held at %s against %s on purpose: %s", module, pin, newest, reason)
			continue
		}

		switch semver.Compare(pin, newest) {
		case 0:
			// The ordinary case.
		case 1:
			// The pin is *ahead* of what the proxy will admit to, which is
			// what a release looks like for the next half hour: the version
			// resolves, and @latest is served from a cache that has not
			// expired. Reading that as a stale pin would fail this test on
			// every release day — including the one that bumps the pin it
			// just published, which is the commit this test exists to
			// accompany.
			//
			// It is a note rather than silence because the other way to get
			// here is a typo for a version that does not exist, and the pin
			// resolution test below is what would catch that.
			t.Logf("note: %s is pinned at %s, ahead of the proxy's cached latest (%s) — "+
				"a tag published in the last half hour looks exactly like this",
				module, pin, newest)
		default:
			t.Errorf("%s is pinned at %s but %s is published.\n"+
				"`lemmego new` resolves this pin, so a stale one hands every new project "+
				"the bugs that release fixed.\nBump it in _scaffold/stubs/versions.json and "+
				"_scaffold/VERSION in the same commit, or record why it is held back in "+
				"deliberateHolds.", module, pin, newest)
		}
	}
}

// latestPublishedVersions asks the module proxy for all of them at once.
//
// reachable separates "the network is not there", which skips the test, from
// "the proxy answered and the answer is a problem", which fails it. Collapsing
// the two would report an unpublished pin — a module nobody can resolve — as a
// skip, which is the one answer that would be actively misleading.
//
// A transport error on *any* module is read as unreachable, because that is
// what being offline looks like, and because the alternative is a test that
// half-reports.
func latestPublishedVersions(modules []string) (answers map[string]proxyAnswer, reachable bool, err error) {
	client := &http.Client{Timeout: 15 * time.Second}

	type result struct {
		module string
		answer proxyAnswer
		// transport is the error that means "no network", kept apart from
		// answer.err, which means "the proxy said something unusable".
		transport error
	}

	results := make(chan result, len(modules))
	var wg sync.WaitGroup
	for _, module := range modules {
		wg.Add(1)
		go func(module string) {
			defer wg.Done()
			version, answered, lookupErr := latestPublishedVersion(client, module)
			r := result{module: module}
			if !answered {
				r.transport = lookupErr
			} else {
				r.answer = proxyAnswer{version: version, err: lookupErr}
			}
			results <- r
		}(module)
	}
	wg.Wait()
	close(results)

	answers = make(map[string]proxyAnswer, len(modules))
	for r := range results {
		if r.transport != nil {
			return nil, false, r.transport
		}
		answers[r.module] = r.answer
	}
	return answers, true, nil
}

// proxyAnswer is what the proxy said about one module: a version, or why its
// answer could not be used.
type proxyAnswer struct {
	version string
	err     error
}

// latestPublishedVersion asks the module proxy about one module.
//
// The path needs no escaping: the proxy wants an uppercase letter encoded as
// !<lower>, and every module here matches github.com/lemmego/[a-z0-9-]+, the
// same shape the import guard above captures. A pin with a capital in its path
// would fail loudly against this rather than silently resolving to the wrong
// module.
func latestPublishedVersion(client *http.Client, module string) (version string, answered bool, err error) {
	resp, err := client.Get(moduleProxy + "/" + module + "/@latest")
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", true, fmt.Errorf("%s/%s/@latest answered %s", moduleProxy, module, resp.Status)
	}

	var latest struct{ Version string }
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return "", true, fmt.Errorf("decoding the proxy's answer: %w", err)
	}
	if latest.Version == "" {
		return "", true, fmt.Errorf("the proxy reported no version")
	}
	return latest.Version, true, nil
}
