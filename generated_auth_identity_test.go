package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// authCombinations is every project the auth overlays can produce: three ORMs,
// each with and without GPA. The overlay directory is chosen from exactly
// these two fields, so this is the whole space.
func authCombinations() []ProjectConfig {
	var configs []ProjectConfig
	for _, orm := range []OrmChoice{OrmLemmego, OrmGORM, OrmBun} {
		for _, gpa := range []bool{false, true} {
			name := "auth" + string(orm)
			if gpa {
				name += "gpa"
			}
			configs = append(configs, ProjectConfig{
				Name:       name,
				ModuleName: "example.com/contracts/" + name,
				Preset:     PresetMVC,
				ORM:        orm,
				SQLDriver:  SQLSQLite,
				EnableAuth: true,
				EnableGPA:  gpa,
				// A queue as well, because auth and the queue together are
				// what emit the dashboard guard — the slices/strings imports
				// and the taskerAdmins helper exist in no other combination,
				// so without this they would never be compiled.
				QueueDriver: QueueSQL,
			})
		}
	}
	return configs
}

// The current user is resolved by re-reading the row each request, so every
// generated project has to carry three things: a loader wired into the auth
// provider, a repository method that takes the string id a credential carries,
// and no gob registration — the session holds an id now, not the object, and
// registering the type invited the old payload back.
//
// These are string checks on generated source rather than a build, so they run
// offline on every `go test`. The build below is the one that would catch a
// type error, and it needs the network.
func TestGeneratedAuthProjectsResolveTheUserFromStorage(t *testing.T) {
	for _, cfg := range authCombinations() {
		t.Run(cfg.Name, func(t *testing.T) {
			root := scaffoldContractProject(t, cfg)

			providers := readGenerated(t, root, "bootstrap/providers.go")
			if !strings.Contains(providers, "UserLoader:") {
				t.Error("the auth provider does not configure a UserLoader, so auth cannot resolve a user at all")
			}
			if !strings.Contains(providers, ".FindByID(c.RequestContext(), id)") {
				t.Error("the loader does not call FindByID with the id auth hands it")
			}

			repo := readGenerated(t, root, "internal/repos/user_repo.go")
			if !strings.Contains(repo, "func (r *UserRepository) FindByID(ctx context.Context, id string)") {
				t.Errorf("the user repository has no FindByID taking a string id:\n%s", repo)
			}
			// A stale or fabricated id must read as "nobody", because the
			// alternative is answering 503 and turning an upgrade into an
			// apparent outage.
			if !strings.Contains(repo, "auth.ErrUserNotFound") {
				t.Error("FindByID does not map its misses onto auth.ErrUserNotFound")
			}

			user := readGenerated(t, root, "internal/models/user.go")
			if strings.Contains(user, "gob.Register") {
				t.Error("the user model still registers itself with gob: the session stores an id, not the user")
			}
		})
	}
}

// The API's /me must read the application's own type. Going through
// auth.AuthUser hands back an any, which is how the four-shaped union this
// design removes got into handlers in the first place.
func TestGeneratedAPIReadsTheUserWithATypedAccessor(t *testing.T) {
	cfg := authCombinations()[0]
	root := scaffoldContractProject(t, cfg)

	routes := readGenerated(t, root, "internal/routes/api.go")
	if !strings.Contains(routes, "auth.UserAs[*models.User](c)") {
		t.Errorf("/me does not use the typed accessor:\n%s", routes)
	}
	if strings.Contains(routes, "auth.AuthUser(") {
		t.Error("/me still reads the untyped user")
	}
}

// Parsing and gofmt cannot see a type error, which is how auth_bun_gpa shipped
// for as long as it did: its repository embedded gpa.MigratableRepository
// while the provider returned gpa.SQLRepository, and every offline check
// passed. Only a compiler catches that.
//
// It is gated because it downloads each project's dependencies. Run it before
// releasing the scaffold:
//
//	LEMMEGO_SCAFFOLD_BUILD=1 go test -run TestGeneratedAuthProjectsCompile -timeout 30m ./...
//
// Point LEMMEGO_SCAFFOLD_REPLACE at a checkout of the monorepo to build
// against the working tree instead of the pinned releases, which is the only
// way to run this before the modules it depends on are tagged:
//
//	LEMMEGO_SCAFFOLD_BUILD=1 LEMMEGO_SCAFFOLD_REPLACE=$HOME/code/lemmego go test ...
func TestGeneratedAuthProjectsCompile(t *testing.T) {
	if os.Getenv("LEMMEGO_SCAFFOLD_BUILD") == "" {
		t.Skip("set LEMMEGO_SCAFFOLD_BUILD=1 to build the generated projects")
	}
	gocmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go toolchain on PATH")
	}

	for _, cfg := range authCombinations() {
		t.Run(cfg.Name, func(t *testing.T) {
			t.Parallel()
			root := scaffoldContractProject(t, cfg)
			if replace := os.Getenv("LEMMEGO_SCAFFOLD_REPLACE"); replace != "" {
				replaceLocalModules(t, gocmd, root, replace)
			}

			// vet rather than build: it type-checks every package, main
			// included, which is the whole point here, and it skips the link
			// step — linking six projects wants a few hundred megabytes of
			// scratch space that a working machine may not have spare.
			//
			// GOWORK=off: a workspace above the temporary directory would
			// resolve these modules from the working tree and hide exactly the
			// mismatch this test exists to find.
			for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}} {
				cmd := exec.Command(gocmd, args...)
				cmd.Dir = root
				cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
				}
			}
		})
	}
}

// replaceLocalModules points every github.com/lemmego/* requirement at the
// corresponding directory in a monorepo checkout, so the generated project
// builds against unreleased code.
func replaceLocalModules(t *testing.T, gocmd, root, monorepo string) {
	t.Helper()
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(gomod), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || !strings.HasPrefix(fields[0], "github.com/lemmego/") {
			continue
		}
		dir := filepath.Join(monorepo, strings.TrimPrefix(fields[0], "github.com/lemmego/"))
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		cmd := exec.Command(gocmd, "mod", "edit", "-replace="+fields[0]+"="+dir)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go mod edit: %v\n%s", err, out)
		}
	}
}

// A project with a queue and authentication gets a dashboard rule it can
// actually use. The rule must call auth.Check itself: the dashboard is
// mounted as a raw http.Handler, so no router middleware runs for it and
// nothing has looked at the cookie by the time the predicate is called.
// Without that call the user is always absent and the dashboard refuses
// everyone, including the people named in TASKER_ADMINS.
func TestGeneratedQueueDashboardIsGuarded(t *testing.T) {
	cfg := authCombinations()[0]
	cfg.QueueDriver = QueueSQL
	root := scaffoldContractProject(t, cfg)

	providers := readGenerated(t, root, "bootstrap/providers.go")
	if !strings.Contains(providers, "DashboardAuth:") {
		t.Fatalf("the queue provider has no DashboardAuth, so the dashboard is unreachable:\n%s", providers)
	}
	if !strings.Contains(providers, "auth.Check(c)") {
		t.Error("DashboardAuth does not call auth.Check, so it will refuse everyone")
	}
	if !strings.Contains(providers, "auth.UserAs[*models.User](c)") {
		t.Error("DashboardAuth does not resolve the application's user type")
	}

	env := readGenerated(t, root, ".env.example")
	if !strings.Contains(env, "TASKER_ADMINS") {
		t.Error("TASKER_ADMINS is not documented in .env.example")
	}
}

// Without authentication there is nobody to recognise, so the dashboard has
// to stay closed rather than fall open.
func TestGeneratedQueueDashboardStaysClosedWithoutAuth(t *testing.T) {
	root := scaffoldContractProject(t, ProjectConfig{
		Name:        "queuenoauth",
		ModuleName:  "example.com/contracts/queuenoauth",
		Preset:      PresetRESTAPI,
		ORM:         OrmLemmego,
		SQLDriver:   SQLSQLite,
		QueueDriver: QueueSQL,
		EnableAuth:  false,
	})

	providers := readGenerated(t, root, "bootstrap/providers.go")
	if strings.Contains(providers, "DashboardAuth:") {
		t.Error("a project with no authentication cannot express a dashboard rule")
	}
	if !strings.Contains(providers, "&queue.Provider{") {
		t.Error("the queue provider is missing entirely")
	}
}
