package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The pins in versions.json that are not ours were guarded by nothing.
//
// TestEveryPinIsTheLatestPublishedVersion opens with a `continue` on anything
// outside github.com/lemmego/, deliberately: we control those releases, so
// "pinned at the latest published version" is a rule we can actually hold. It
// is not a rule anybody can hold against templ, which published 175 patch
// releases in the window this scaffold sat still.
//
// So the third-party half drifted exactly as the lemmego half had before that
// test existed, and by more: when this file was written every one of the nine
// was between 435 and 1073 days behind its newest release. vite was pinned at
// 7.1.4 against 8.3.8, typescript at ^5.4.5 against 7.0.2 — a major behind,
// which a caret range cannot cross — and lib/pq had been still for 1073 days.
//
// The two tests below therefore ask a weaker question than the lemmego one,
// and ask it of the two ecosystems separately, because what is cheap to know
// differs between them.

// thirdPartyStalenessBudget is how far behind its newest release a third-party
// Go pin may be before this is drift rather than a version choice.
//
// Six months is deliberately generous. The point is not to track upstream: it
// is to catch a pin nobody has looked at in over a year, which is what all
// five Go pins were. A budget tight enough to fire on a patch released last
// week would fire constantly, and a test that fires constantly gets deleted —
// the failure mode TestEveryPinIsTheLatestPublishedVersion's own comment
// records about breaking `go test` on a plane. The drift this found was 2.4x
// to 6x this budget, so there is no need to be clever about the number.
const thirdPartyStalenessBudget = 180 * 24 * time.Hour

// TestEveryThirdPartyGoPinIsRecentEnough holds the Go pins we do not publish
// to a staleness budget rather than to the latest version.
//
// The proxy serves a publish date for an exact version and for the newest one,
// both as a few hundred bytes, so the budget is affordable here in a way it is
// not for npm — see the next test.
func TestEveryThirdPartyGoPinIsRecentEnough(t *testing.T) {
	if os.Getenv("GOPROXY") == "off" {
		t.Skip("GOPROXY=off, so there is nothing to ask for publish dates")
	}

	pins := scaffoldPins(t)

	var modules []string
	for module := range pins {
		if !strings.HasPrefix(module, "github.com/") && !strings.HasPrefix(module, "golang.org/") &&
			!strings.HasPrefix(module, "gorm.io/") && !strings.HasPrefix(module, "google.golang.org/") {
			continue // an npm package; the next test owns it
		}
		if strings.HasPrefix(module, "github.com/lemmego/") {
			continue // ours, and held to the stricter rule
		}
		modules = append(modules, module)
	}
	sort.Strings(modules)
	if len(modules) == 0 {
		t.Fatal("versions.json pins no third-party Go module, so this test proves nothing")
	}

	client := &http.Client{Timeout: 20 * time.Second}

	type dated struct {
		newestVersion string
		newestAt      time.Time
		pinnedAt      time.Time
		err           error
		transport     error
	}

	answers := make(map[string]dated, len(modules))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, module := range modules {
		wg.Add(1)
		go func(module string) {
			defer wg.Done()
			var d dated
			newestVersion, newestAt, transport, err := goModuleRelease(client, module, "@latest")
			switch {
			case transport != nil:
				d.transport = transport
			case err != nil:
				d.err = err
			default:
				d.newestVersion, d.newestAt = newestVersion, newestAt
				_, pinnedAt, transport, err := goModuleRelease(client, module, "@v/"+pins[module]+".info")
				switch {
				case transport != nil:
					d.transport = transport
				case err != nil:
					d.err = err
				default:
					d.pinnedAt = pinnedAt
				}
			}
			mu.Lock()
			answers[module] = d
			mu.Unlock()
		}(module)
	}
	wg.Wait()

	// One unreachable module skips the whole test, for the reason the lemmego
	// test gives: a partial answer reported as green is how a vacuous pass
	// happens.
	for _, module := range modules {
		if transport := answers[module].transport; transport != nil {
			t.Skipf("cannot reach %s (%v).\nThis test needs the network; CI runs it.",
				moduleProxy, transport)
		}
	}

	for _, module := range modules {
		a := answers[module]
		if a.err != nil {
			t.Errorf("%s is pinned at %s but the proxy could not be asked about it: %v\n"+
				"A pin that cannot be resolved is a project that cannot be generated.",
				module, pins[module], a.err)
			continue
		}
		behind := a.newestAt.Sub(a.pinnedAt)
		if reason, held := deliberateHolds[module]; held {
			t.Logf("note: %s held at %s against %s on purpose: %s", module, pins[module], a.newestVersion, reason)
			continue
		}
		if behind <= thirdPartyStalenessBudget {
			if pins[module] != a.newestVersion {
				t.Logf("note: %s is pinned at %s against %s, %d days behind and within budget",
					module, pins[module], a.newestVersion, int(behind.Hours()/24))
			}
			continue
		}
		t.Errorf("%s is pinned at %s, which is %d days older than %s.\n"+
			"That is past the %d-day budget, so it is drift rather than a choice: "+
			"`lemmego new` hands every project this version.\nBump it in "+
			"_scaffold/stubs/versions.json and _scaffold/VERSION in the same commit, "+
			"or record why it is held back in deliberateHolds.",
			module, pins[module], int(behind.Hours()/24), a.newestVersion,
			int(thirdPartyStalenessBudget.Hours()/24))
	}
}

// TestEveryNpmPinIsOnTheCurrentMajor holds the npm pins to their major version
// and no finer, which is as much as is both affordable and meaningful.
//
// Affordable: npm publishes a per-version date only inside the full packument,
// which is 16MB for typescript and 38MB for vite. Twenty of those is not a
// test. The cheap endpoints — /<name>/latest and /<name>/<version> — carry no
// date at all, so the budget the Go test uses cannot be computed here.
//
// Meaningful: every pin in this table is a caret range, so `npm install`
// already resolves to the newest release inside the major. react pinned at
// ^19.0.0 installs 19.3.1 today and needs no bump. A major is the one gap a
// range cannot cross by itself, and it is what actually bit — typescript sat
// at ^5.4.5 while 7.0.2 was current, and vite at 7.1.4 against 8.3.8 while
// laravel-vite-plugin 3 and @vitejs/plugin-react 6 both require vite ^8.
//
// The limitation is worth stating rather than hiding: an *exact* npm pin that
// falls behind within its major would pass this test. vite was the only one,
// and it is a range now.
func TestEveryNpmPinIsOnTheCurrentMajor(t *testing.T) {
	if os.Getenv("GOPROXY") == "off" {
		t.Skip("GOPROXY=off is taken to mean no network, so the registry is unreachable too")
	}

	pins := scaffoldPins(t)

	var packages []string
	for name := range pins {
		if strings.Contains(name, ".") && strings.Contains(name, "/") {
			continue // a module path, not an npm name
		}
		packages = append(packages, name)
	}
	sort.Strings(packages)
	if len(packages) == 0 {
		t.Fatal("versions.json pins no npm package, so this test proves nothing")
	}

	client := &http.Client{Timeout: 20 * time.Second}

	type answer struct {
		latest    string
		err       error
		transport error
	}
	answers := make(map[string]answer, len(packages))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, name := range packages {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			latest, transport, err := npmLatestVersion(client, name)
			mu.Lock()
			answers[name] = answer{latest: latest, err: err, transport: transport}
			mu.Unlock()
		}(name)
	}
	wg.Wait()

	for _, name := range packages {
		if transport := answers[name].transport; transport != nil {
			t.Skipf("cannot reach registry.npmjs.org (%v).\nThis test needs the network; CI runs it.", transport)
		}
	}

	for _, name := range packages {
		a := answers[name]
		if a.err != nil {
			t.Errorf("%s is pinned at %s but the registry could not be asked about it: %v",
				name, pins[name], a.err)
			continue
		}
		pinnedMajor, ok := semverMajor(pins[name])
		if !ok {
			t.Errorf("%s is pinned at %q, which this test cannot read as a version range.\n"+
				"Pin it as ^x.y.z or x.y.z so the major is legible.", name, pins[name])
			continue
		}
		latestMajor, ok := semverMajor(a.latest)
		if !ok {
			t.Errorf("%s: the registry's latest version %q is unreadable", name, a.latest)
			continue
		}
		if reason, held := deliberateHolds[name]; held {
			t.Logf("note: %s held at %s against %s on purpose: %s", name, pins[name], a.latest, reason)
			continue
		}
		if pinnedMajor == latestMajor {
			continue
		}
		if pinnedMajor > latestMajor {
			// A prerelease major, or a pin made minutes ago against a tag the
			// registry has not promoted to `latest`. Noted rather than failed,
			// the way the lemmego test treats a pin ahead of the proxy.
			t.Logf("note: %s is pinned at %s, ahead of the registry's latest (%s)",
				name, pins[name], a.latest)
			continue
		}
		t.Errorf("%s is pinned at %s but %s is published, a major ahead.\n"+
			"A caret range cannot cross a major, so every generated project is held "+
			"at %d.x until this is bumped.\nBump it in _scaffold/stubs/versions.json "+
			"and _scaffold/VERSION in the same commit, or record why it is held back "+
			"in deliberateHolds.", name, pins[name], a.latest, pinnedMajor)
	}
}

// goModuleRelease asks the proxy about one version of one module and returns
// its version and publish date.
//
// transport separates "no network" from "the proxy said something unusable",
// which is the distinction that decides between skipping and failing.
func goModuleRelease(client *http.Client, module, suffix string) (version string, published time.Time, transport, err error) {
	resp, getErr := client.Get(moduleProxy + "/" + escapeModulePath(module) + "/" + suffix)
	if getErr != nil {
		return "", time.Time{}, getErr, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, nil, fmt.Errorf("%s answered %s", suffix, resp.Status)
	}
	var body struct {
		Version string
		Time    time.Time
	}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&body); decodeErr != nil {
		return "", time.Time{}, nil, decodeErr
	}
	if body.Time.IsZero() {
		return "", time.Time{}, nil, fmt.Errorf("%s carried no publish date", suffix)
	}
	return body.Version, body.Time, nil, nil
}

// escapeModulePath applies the proxy's case encoding: an upper-case letter is
// written as "!" followed by its lower-case form, because module paths are
// case-sensitive and the paths they are served from are not.
func escapeModulePath(module string) string {
	var b strings.Builder
	for _, r := range module {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// npmLatestVersion reads the version the registry serves as `latest`.
//
// /<name>/latest is a few kilobytes. The full packument carries publish dates
// and is tens of megabytes, which is why the test above compares majors.
func npmLatestVersion(client *http.Client, name string) (latest string, transport, err error) {
	resp, getErr := client.Get("https://registry.npmjs.org/" + strings.ReplaceAll(url.PathEscape(name), "/", "%2f") + "/latest")
	if getErr != nil {
		return "", getErr, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("registry answered %s", resp.Status)
	}
	var body struct{ Version string }
	if decodeErr := json.NewDecoder(resp.Body).Decode(&body); decodeErr != nil {
		return "", nil, decodeErr
	}
	if body.Version == "" {
		return "", nil, fmt.Errorf("registry returned no version")
	}
	return body.Version, nil, nil
}

var majorPattern = regexp.MustCompile(`(\d+)`)

// semverMajor reads the major version out of a pin, whether it is written as
// an exact version or as a range. "^8.3.8", "8.3.8" and ">=8.0.0" all give 8.
func semverMajor(pin string) (int, bool) {
	match := majorPattern.FindStringSubmatch(pin)
	if match == nil {
		return 0, false
	}
	n, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return n, true
}
