package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestEveryModuleHasAnOverlay.
//
// resolveOverlays builds the path from the module's name, and applyOverlay
// fails loudly on one that is missing — which is right, but it fails during
// `lemmego new`, in front of somebody creating a project. A name added to
// AllModules without its directory is caught here instead.
func TestEveryModuleHasAnOverlay(t *testing.T) {
	for _, spec := range AllModules {
		dir := filepath.Join("_scaffold", "overlays", "mod_"+string(spec.name))

		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			t.Errorf("module %q has no overlay at %s.\nEvery module contributes its "+
				"provider wiring and one worked example; without the directory, "+
				"`lemmego new --modules %s` fails at generation time.",
				spec.name, dir, spec.name)
			continue
		}

		// A directory that exists and contains nothing would scaffold a
		// project that pins the module, imports nothing and shows nobody how
		// to use it — which is worse than not offering it.
		var goFiles int
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".go") {
				goFiles++
			}
			return nil
		})
		if goFiles == 0 {
			t.Errorf("overlay %s contains no Go files", dir)
		}
	}
}

// TestEveryModuleRouteCallbackIsDefined is the forward half of the route
// wiring, and it has a quiet failure to catch: routes_bootstrap.go.tpl lists
// what ModuleRoutes returns, so a name that no overlay defines does not
// compile — but only once somebody generates a project with that module and
// builds it.
func TestEveryModuleRouteCallbackIsDefined(t *testing.T) {
	for module, callback := range moduleRouteCallbacks() {
		dir := filepath.Join("_scaffold", "overlays", "mod_"+string(module))
		want := regexp.MustCompile(`func ` + callback + `\(a app\.App\)`)

		if !treeMatches(t, dir, want) {
			t.Errorf("ModuleRoutes names routes.%s for module %q, but no file under "+
				"%s defines `func %s(a app.App)`.\nThe generated bootstrap/routes.go "+
				"would not compile.", callback, module, dir, callback)
		}
	}
}

// TestEveryModuleRouteFileIsRegistered is the other half, and the one that
// fails silently rather than loudly.
//
// An overlay that defines a route callback nobody names in ModuleRoutes
// compiles perfectly and mounts nothing. The example is then dead code in
// somebody's project, and the symptom is a 404 on a path the README promises.
func TestEveryModuleRouteFileIsRegistered(t *testing.T) {
	declared := moduleRouteCallbacks()
	defined := regexp.MustCompile(`func ([A-Z][A-Za-z]*Routes)\(a app\.App\)`)

	for _, spec := range AllModules {
		dir := filepath.Join("_scaffold", "overlays", "mod_"+string(spec.name))

		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for _, match := range defined.FindAllStringSubmatch(string(raw), -1) {
				if declared[spec.name] != match[1] {
					t.Errorf("%s defines %s, which ModuleRoutes does not name for "+
						"module %q.\nIt would be compiled and never mounted, so the "+
						"example answers 404.", path, match[1], spec.name)
				}
			}
			return nil
		})
	}
}

// TestAModuleWhoseRequirementsAreUnmetIsRejected.
//
// The interactive form withdraws these rather than offering them, so this is
// about the other two ways a configuration arrives: --modules on the command
// line, and a ProjectConfig written in Go.
//
// Each case is a project that would scaffold cleanly and then fail — at boot
// for the ones missing a service, and inside a migration for the two dialect
// cases.
func TestAModuleWhoseRequirementsAreUnmetIsRejected(t *testing.T) {
	base := func() ProjectConfig {
		cfg := defaultProjectConfig("demo")
		cfg.ModuleName = "example.com/demo"
		return cfg
	}

	cases := []struct {
		name   string
		module Module
		mutate func(*ProjectConfig)
		want   string
	}{
		{
			name:   "authz without authentication",
			module: ModuleAuthz,
			mutate: func(c *ProjectConfig) { c.EnableAuth = false },
			want:   "authz needs authentication",
		},
		{
			name:   "ratelimit without a cache",
			module: ModuleRatelimit,
			mutate: func(c *ProjectConfig) { c.CacheDriver = CacheNone },
			want:   "ratelimit needs a cache",
		},
		{
			name:   "notify without a database",
			module: ModuleNotify,
			// The queue and session drivers go too, or validate refuses the
			// configuration for one of those first and this case would pass
			// on somebody else's error message.
			mutate: func(c *ProjectConfig) {
				c.SQLDriver = SQLNone
				c.EnableAuth = false
				c.QueueDriver = QueueNone
				c.SessionDriver = SessionFile
			},
			want: "notify needs a database",
		},
		{
			// notify's own store ships statements for postgres and sqlite
			// and refuses anything else, so this would fail inside the
			// migration rather than at boot.
			name:   "notify on mysql",
			module: ModuleNotify,
			mutate: func(c *ProjectConfig) { c.SQLDriver = SQLMySQL },
			want:   "notify has no MySQL store",
		},
		{
			name:   "pay on mysql",
			module: ModulePay,
			mutate: func(c *ProjectConfig) { c.SQLDriver = SQLMySQL },
			want:   "pay has no MySQL store",
		},
		{
			// The default engine searches tsvector columns.
			name:   "search on sqlite",
			module: ModuleSearch,
			mutate: func(c *ProjectConfig) { c.SQLDriver = SQLSQLite },
			want:   "search needs postgres",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.mutate(&cfg)
			cfg.Modules = []Module{tc.module}

			normalize(&cfg)
			err := validate(cfg)
			if err == nil {
				t.Fatalf("validate accepted %s with %s, which would scaffold a "+
					"project that cannot run", tc.module, tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("the refusal does not say why.\n got: %v\nwant it to mention: %s",
					err, tc.want)
			}
		})
	}
}

// TestAModuleWhoseRequirementsAreMetIsAccepted, so the test above is not
// passing because validate refuses everything.
func TestAModuleWhoseRequirementsAreMetIsAccepted(t *testing.T) {
	cfg := defaultProjectConfig("demo")
	cfg.ModuleName = "example.com/demo"
	cfg.SQLDriver = SQLPostgres
	cfg.ORM = OrmLemmego
	cfg.CacheDriver = CacheFile
	cfg.EnableAuth = true
	for _, spec := range AllModules {
		cfg.Modules = append(cfg.Modules, spec.name)
	}

	normalize(&cfg)
	if err := validate(cfg); err != nil {
		t.Fatalf("a postgres project with auth and a cache should support every "+
			"module, and validate refused: %v", err)
	}
	if len(cfg.Modules) != len(AllModules) {
		t.Errorf("normalize dropped modules: kept %d of %d", len(cfg.Modules), len(AllModules))
	}
}

// TestAllMeansEveryModuleThisProjectCanSupport.
//
// Not every module that exists, which is the other reading and the useless
// one: `--modules all` on a project with no database would then be an error
// rather than "everything I can have".
func TestAllMeansEveryModuleThisProjectCanSupport(t *testing.T) {
	cfg := defaultProjectConfig("demo")
	cfg.SQLDriver = SQLSQLite
	cfg.ORM = OrmLemmego
	cfg.CacheDriver = CacheNone
	cfg.EnableAuth = false

	modules, err := ParseModules("all", cfg)
	if err != nil {
		t.Fatalf("all: %v", err)
	}

	got := map[Module]bool{}
	for _, m := range modules {
		got[m] = true
	}

	// Withheld: authz needs auth, ratelimit needs the cache, search needs
	// postgres.
	for _, absent := range []Module{ModuleAuthz, ModuleRatelimit, ModuleSearch} {
		if got[absent] {
			t.Errorf("all included %s on a project that cannot support it", absent)
		}
	}
	// Present: these need nothing this project lacks.
	for _, want := range []Module{ModuleSeo, ModuleMail, ModuleMedia, ModuleI18n, ModuleObserve, ModuleNotify} {
		if !got[want] {
			t.Errorf("all left out %s, which this project supports", want)
		}
	}

	cfg.Modules = modules
	if err := validate(cfg); err != nil {
		t.Errorf("all produced a configuration validate refuses: %v", err)
	}
}

// TestAnUnknownModuleIsNamedRatherThanIgnored.
func TestAnUnknownModuleIsNamedRatherThanIgnored(t *testing.T) {
	_, err := ParseModules("seo,telemetry", defaultProjectConfig("demo"))
	if err == nil {
		t.Fatal("--modules telemetry was accepted")
	}
	if !strings.Contains(err.Error(), "telemetry") || !strings.Contains(err.Error(), "seo") {
		t.Errorf("the error should name the unknown module and list the valid "+
			"ones; got: %v", err)
	}
}

// TestModulesAreSortedAndDeduplicated.
//
// resolveOverlays walks this slice in order, so two identical configurations
// have to produce the same overlay order — otherwise two runs of the same
// command could differ wherever two overlays touch one path.
func TestModulesAreSortedAndDeduplicated(t *testing.T) {
	cfg := defaultProjectConfig("demo")
	cfg.SQLDriver = SQLPostgres
	cfg.Modules = []Module{ModuleSeo, ModuleMail, ModuleSeo, ModuleI18n}

	normalizeModules(&cfg)

	want := []Module{ModuleI18n, ModuleMail, ModuleSeo}
	if !reflect.DeepEqual(cfg.Modules, want) {
		t.Errorf("normalizeModules\n got: %v\nwant: %v", cfg.Modules, want)
	}
}

// TestModuleOverlaysAreAppliedAfterTheAuthOverlay.
//
// A later overlay wins where two name one path, and a module overlay may
// refer to the user model the auth overlay writes. The order is also what
// makes the result of two identical runs identical.
func TestModuleOverlaysAreAppliedAfterTheAuthOverlay(t *testing.T) {
	cfg := defaultProjectConfig("demo")
	cfg.SQLDriver = SQLPostgres
	cfg.ORM = OrmLemmego
	cfg.EnableAuth = true
	cfg.Modules = []Module{ModuleSeo, ModuleAuthz}

	overlays := resolveOverlays(cfg)

	authAt, firstModuleAt := -1, -1
	for i, overlay := range overlays {
		if strings.HasPrefix(overlay, "overlays/auth_") {
			authAt = i
		}
		if strings.HasPrefix(overlay, "overlays/mod_") && firstModuleAt == -1 {
			firstModuleAt = i
		}
	}

	if authAt == -1 {
		t.Fatal("no auth overlay was resolved for a project with authentication")
	}
	if firstModuleAt == -1 {
		t.Fatal("no module overlay was resolved")
	}
	if firstModuleAt < authAt {
		t.Errorf("module overlays are applied before the auth overlay: %v", overlays)
	}
}

// TestEveryModuleIsPinnedAtItsOwnName.
//
// go.mod.tpl renders one require line per selected module from ModulePins,
// which builds the path from the module's name — so a name that is not also
// the last element of its import path would pin the wrong module, or pin
// nothing and resolve to "latest".
func TestEveryModuleIsPinnedAtItsOwnName(t *testing.T) {
	pins := scaffoldPins(t)

	for _, spec := range AllModules {
		path := ModuleImportPath(spec.name)
		if _, ok := pins[path]; !ok {
			t.Errorf("module %q resolves to the import path %s, which "+
				"versions.json does not pin.\nVersion would answer \"latest\", "+
				"which is the pinning policy defeated in the one file whose job "+
				"is to pin.", spec.name, path)
		}
	}
}

// moduleRouteCallbacks reads the map ModuleRoutes consults, by asking it.
//
// Driven through the real method rather than duplicating the table, so a
// module added to one and not the other is caught instead of being described
// twice.
func moduleRouteCallbacks() map[Module]string {
	callbacks := map[Module]string{}
	for _, spec := range AllModules {
		cfg := defaultProjectConfig("demo")
		cfg.Modules = []Module{spec.name}
		td := templateData{ProjectConfig: cfg}

		if names := td.ModuleRoutes(); len(names) == 1 {
			callbacks[spec.name] = names[0]
		}
	}
	return callbacks
}

func treeMatches(t *testing.T, dir string, want *regexp.Regexp) bool {
	t.Helper()

	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr == nil && want.Match(raw) {
			found = true
		}
		return nil
	})
	return found
}

// TestModuleNamesAreStable guards the one thing every other part of this
// feature keys on.
//
// The overlay directory, the import path, the .env block, the --modules value
// and the picker label are all derived from this string. Renaming one is a
// breaking change for anybody's scripted `lemmego new`, so the list is pinned
// here and a change has to be deliberate.
func TestModuleNamesAreStable(t *testing.T) {
	want := []string{
		"authz", "broadcast", "i18n", "mail", "media", "notify",
		"observe", "pay", "ratelimit", "search", "seo",
	}

	got := ModuleNames()
	sort.Strings(got)
	sorted := append([]string(nil), want...)
	sort.Strings(sorted)

	if !reflect.DeepEqual(got, sorted) {
		t.Errorf("the module list changed\n got: %v\nwant: %v", got, sorted)
	}
}
