package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// assertNoUnusedImports reports an import nothing in the file references.
//
// This is the failure mode pruning a provider produces, and nothing else
// catches it: gofmt only parses, so a file whose imports are all unused
// formats cleanly and passes every other check here. It surfaces as
// "imported and not used" in the user's new project, pointing at generated
// code they did not write.
//
// Matching identifiers rather than type-checking keeps the test offline, which
// the scaffold tests are required to be.
func assertNoUnusedImports(t *testing.T, path string, file *ast.File) {
	t.Helper()

	used := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		selector, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := selector.X.(*ast.Ident); ok && ident.Obj == nil {
			used[ident.Name] = true
		}
		return true
	})

	for _, imported := range file.Imports {
		// Side-effect and dot imports are unreferenced by design.
		if imported.Name != nil && (imported.Name.Name == "_" || imported.Name.Name == ".") {
			continue
		}
		if !used[importedName(imported)] {
			t.Errorf("%s imports %s but never uses it", path, imported.Path.Value)
		}
	}
}

// importedName is the identifier a package is referenced by: its alias, or the
// last element of its path.
func importedName(imported *ast.ImportSpec) string {
	if imported.Name != nil {
		return imported.Name.Name
	}
	path, err := strconv.Unquote(imported.Path.Value)
	if err != nil {
		return ""
	}
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// Every combination a user can reach must produce Go that parses and has no
// imports left over. The rows are chosen so each covers something no other row
// does, rather than to enumerate the product.
func TestGeneratedProjectsCompileCleanly(t *testing.T) {
	tests := []struct {
		name string
		cfg  ProjectConfig
	}{
		{
			// What most people get.
			name: "defaults",
			cfg:  ProjectConfig{ModuleName: "example.com/defaults"},
		},
		{
			// The minimum viable project: no database, cache or queue. This is
			// the row the None option exists for.
			name: "everything off",
			cfg: ProjectConfig{
				ModuleName: "example.com/minimal", Preset: PresetRESTAPI,
				SQLDriver: SQLNone, CacheDriver: CacheNone, QueueDriver: QueueNone,
				SessionDriver: SessionMemory,
			},
		},
		{
			// Everything on Redis, plus S3 and Postgres.
			name: "all redis",
			cfg: ProjectConfig{
				ModuleName: "example.com/redis", Preset: PresetRESTAPI,
				SQLDriver: SQLPostgres, ORM: OrmGORM, CacheDriver: CacheRedis,
				QueueDriver: QueueRedis, SessionDriver: SessionRedis, FilesystemDisk: DiskS3,
			},
		},
		{
			// Auth, GPA, Bun and MySQL — the connector path that panicked on a
			// missing options key.
			name: "bun mysql with auth and gpa",
			cfg: ProjectConfig{
				ModuleName: "example.com/bun", Preset: PresetMVC, Frontend: FrontendTempl,
				SQLDriver: SQLMySQL, ORM: OrmBun, CacheDriver: CacheMemory,
				EnableAuth: true, EnableGPA: true,
			},
		},
		{
			// Redis for sessions only: the shared connection block must be
			// written even though neither the cache nor the queue asked for it.
			name: "redis for sessions only",
			cfg: ProjectConfig{
				ModuleName: "example.com/sess", Preset: PresetMVC, Frontend: FrontendInertiaReact,
				SessionDriver: SessionRedis, CacheDriver: CacheFile, EnableAuth: true,
			},
		},
		{
			// No database, but a Redis queue: the queue must not need one.
			name: "no database with a redis queue",
			cfg: ProjectConfig{
				ModuleName: "example.com/noqdb", Preset: PresetRESTAPI,
				SQLDriver: SQLNone, CacheDriver: CacheNone, QueueDriver: QueueRedis,
			},
		},
		{
			name: "vue with the lemmego orm",
			cfg: ProjectConfig{
				ModuleName: "example.com/vue", Preset: PresetMVC, Frontend: FrontendInertiaVue,
				SQLDriver: SQLSQLite, ORM: OrmLemmego, EnableAuth: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := ScaffoldProject(tt.cfg, dir); err != nil {
				t.Fatalf("ScaffoldProject() error = %v", err)
			}

			fset := token.NewFileSet()
			seen := 0
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || filepath.Ext(path) != ".go" {
					return err
				}
				seen++

				source, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if strings.Contains(string(source), "{{") {
					t.Errorf("%s contains an unrendered template action", path)
				}

				file, err := parser.ParseFile(fset, path, source, parser.AllErrors)
				if err != nil {
					t.Errorf("%s does not parse: %v", path, err)
					return nil
				}
				assertNoUnusedImports(t, path, file)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if seen == 0 {
				t.Fatal("no Go files were generated")
			}
		})
	}
}

// A subsystem left out must leave nothing behind: not the provider, not the
// dependency, not the configuration file, not the environment variables.
func TestOmittedSubsystemsLeaveNothingBehind(t *testing.T) {
	dir := t.TempDir()
	cfg := ProjectConfig{
		ModuleName: "example.com/minimal", Preset: PresetRESTAPI,
		SQLDriver: SQLNone, CacheDriver: CacheNone, QueueDriver: QueueNone,
		SessionDriver: SessionMemory,
	}
	if err := ScaffoldProject(cfg, dir); err != nil {
		t.Fatal(err)
	}

	for _, absent := range []string{
		"internal/configs/database.go",
		"internal/configs/cache.go",
		"internal/configs/tasker.go",
		"internal/configs/keyvalue.go",
	} {
		if _, err := os.Stat(filepath.Join(dir, absent)); err == nil {
			t.Errorf("%s was written for a project that asked for none of it", absent)
		}
	}

	assertFileLacks(t, dir, "bootstrap/providers.go",
		"cache.Provider", "queue.Provider", "ormconnector", "gormconnector", "bunconnector")
	assertFileLacks(t, dir, "go.mod",
		"lemmego/cache", "lemmego/queue", "lemmego/gpa", "lemmego/orm")
	assertFileLacks(t, dir, ".env.example", "DB_", "CACHE_", "TASKER_", "REDIS_")
}

// The shared Redis connection is written exactly when something reads it. The
// helper it defines and the callers of that helper are gated on the same
// predicate, and this is what keeps the two from drifting apart.
func TestTheRedisBlockIsWrittenExactlyWhenItIsRead(t *testing.T) {
	tests := []struct {
		name string
		cfg  ProjectConfig
	}{
		{"nothing on redis", ProjectConfig{ModuleName: "example.com/a"}},
		{"cache on redis", ProjectConfig{ModuleName: "example.com/b", CacheDriver: CacheRedis}},
		{"queue on redis", ProjectConfig{ModuleName: "example.com/c", QueueDriver: QueueRedis}},
		{"session on redis", ProjectConfig{ModuleName: "example.com/d", SessionDriver: SessionRedis}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := ScaffoldProject(tt.cfg, dir); err != nil {
				t.Fatal(err)
			}

			_, statErr := os.Stat(filepath.Join(dir, "internal/configs/keyvalue.go"))
			defined := statErr == nil
			referenced := treeContains(t, dir, "redisAddr()")

			// keyvalue.go both defines the helper and is the only file that
			// can, so a reference without it would not compile.
			if referenced && !defined {
				t.Error("redisAddr() is referenced but the file defining it was not written")
			}
			if defined && !tt.cfg.UsesRedis() {
				t.Error("the Redis block was written for a project that uses no Redis")
			}
			if !defined && tt.cfg.UsesRedis() {
				t.Error("the Redis block was not written for a project that uses Redis")
			}
		})
	}
}

func assertFileLacks(t *testing.T, dir, name string, forbidden ...string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	for _, needle := range forbidden {
		if strings.Contains(string(body), needle) {
			t.Errorf("%s mentions %q, which this project left out", name, needle)
		}
	}
}

func treeContains(t *testing.T, dir, needle string) bool {
	t.Helper()
	found := false
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found {
			return err
		}
		body, readErr := os.ReadFile(path)
		if readErr == nil && strings.Contains(string(body), needle) {
			found = true
		}
		return nil
	})
	return found
}

// Providers run one at a time in the order LoadProviders lists them, and a
// provider can only resolve services the ones before it registered. The
// database connector publishes the connection under db.Connection, so it has
// to precede anything that might want to store something in it — the queue
// above all, which used to open a second pool of its own precisely because
// there was nothing to resolve.
//
// This is invisible at compile time and only shows up as a subsystem quietly
// falling back at runtime, so it is pinned here.
func TestConnectorPrecedesItsConsumers(t *testing.T) {
	for _, tt := range []struct {
		name string
		cfg  ProjectConfig
	}{
		{"lemmego orm", ProjectConfig{
			ModuleName: "example.com/order-orm", SQLDriver: SQLSQLite, ORM: OrmLemmego,
			CacheDriver: CacheMemory, QueueDriver: QueueSQL,
		}},
		{"gorm with gpa", ProjectConfig{
			ModuleName: "example.com/order-gorm", SQLDriver: SQLPostgres, ORM: OrmGORM,
			EnableGPA: true, CacheDriver: CacheRedis, QueueDriver: QueueRedis,
		}},
		{"bun", ProjectConfig{
			ModuleName: "example.com/order-bun", SQLDriver: SQLMySQL, ORM: OrmBun,
			CacheDriver: CacheFile, QueueDriver: QueueSQL,
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := ScaffoldProject(tt.cfg, dir); err != nil {
				t.Fatalf("ScaffoldProject() error = %v", err)
			}
			source, err := os.ReadFile(filepath.Join(dir, "bootstrap", "providers.go"))
			if err != nil {
				t.Fatal(err)
			}
			providers := string(source)

			connector := -1
			for _, name := range []string{"ormconnector.Provider", "gormconnector.Provider", "bunconnector.Provider"} {
				if at := strings.Index(providers, "&"+name); at >= 0 {
					connector = at
					break
				}
			}
			if connector < 0 {
				t.Fatal("no database connector was written despite a SQL driver being selected")
			}

			for _, consumer := range []string{"&queue.Provider{", "&cache.Provider{"} {
				at := strings.Index(providers, consumer)
				if at < 0 {
					t.Fatalf("%s was not written", consumer)
				}
				if at < connector {
					t.Errorf("%s is listed before the database connector; it cannot resolve db.Connection", consumer)
				}
			}
		})
	}
}
