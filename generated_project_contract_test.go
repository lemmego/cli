package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// These checks intentionally stop at parsing and formatting. Compiling a
// generated project would require downloading its application dependencies;
// the embedded scaffold itself is the offline contract under test here.
func TestGeneratedProjectsHaveValidGoStructure(t *testing.T) {
	tests := []struct {
		name string
		cfg  ProjectConfig
	}{
		{
			name: "rest api with gorm",
			cfg: ProjectConfig{
				Name:       "contractapi",
				ModuleName: "example.com/contracts/api",
				Preset:     PresetRESTAPI,
				ORM:        OrmGORM,
			},
		},
		{
			name: "mvc react with auth",
			cfg: ProjectConfig{
				Name:       "contractreact",
				ModuleName: "example.com/contracts/react",
				Preset:     PresetMVC,
				ORM:        OrmGORM,
				EnableAuth: true,
				Frontend:   FrontendInertiaReact,
			},
		},
		{
			name: "mvc vue with bun gpa",
			cfg: ProjectConfig{
				Name:       "contractvue",
				ModuleName: "example.com/contracts/vue",
				Preset:     PresetMVC,
				ORM:        OrmBun,
				EnableAuth: true,
				EnableGPA:  true,
				Frontend:   FrontendInertiaVue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := scaffoldContractProject(t, tt.cfg)
			assertGeneratedGoFiles(t, root)
			assertGeneratedModule(t, root, tt.cfg.ModuleName)
		})
	}
}

func TestGeneratedFrontendEntriesMatchRoutesAndPages(t *testing.T) {
	tests := []struct {
		name       string
		frontend   FrontendPreset
		entry      string
		page       string
		unexpected string
	}{
		{
			name:       "react",
			frontend:   FrontendInertiaReact,
			entry:      "resources/js/app.tsx",
			page:       "IndexReact",
			unexpected: "IndexVue",
		},
		{
			name:       "vue",
			frontend:   FrontendInertiaVue,
			entry:      "resources/js/app.js",
			page:       "IndexVue",
			unexpected: "IndexReact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ProjectConfig{
				Name:       "contract" + tt.name,
				ModuleName: "example.com/contracts/" + tt.name,
				Preset:     PresetMVC,
				ORM:        OrmGORM,
				Frontend:   tt.frontend,
			}
			root := scaffoldContractProject(t, cfg)

			rootHTML := readScaffoldFile(t, root, "resources/views/root.html")
			if !strings.Contains(rootHTML, tt.entry) {
				t.Fatalf("root template does not reference %s", tt.entry)
			}
			if strings.Contains(rootHTML, "app.tsx") != (tt.frontend == FrontendInertiaReact) {
				t.Fatalf("root template has the wrong JavaScript entry: %s", rootHTML)
			}

			routes := readScaffoldFile(t, root, "internal/routes/web.go")
			if !strings.Contains(routes, `"`+tt.page+`"`) || strings.Contains(routes, tt.unexpected) {
				t.Fatalf("routes do not select %s: %s", tt.page, routes)
			}
			extension := ".vue"
			if tt.frontend == FrontendInertiaReact {
				extension = ".tsx"
			}
			pagePath := filepath.Join(root, "resources/js/Pages", tt.page+extension)
			if _, err := os.Stat(pagePath); err != nil {
				t.Fatalf("expected frontend page %s: %v", pagePath, err)
			}
		})
	}
}

func TestGeneratedNodeManifestMatchesFrontendPreset(t *testing.T) {
	for _, frontend := range []FrontendPreset{FrontendInertiaReact, FrontendInertiaVue} {
		t.Run(string(frontend), func(t *testing.T) {
			cfg := ProjectConfig{
				Name:       "contract-node",
				ModuleName: "example.com/contracts/node",
				Preset:     PresetMVC,
				ORM:        OrmGORM,
				Frontend:   frontend,
			}
			root := scaffoldContractProject(t, cfg)

			var packageJSON struct {
				Scripts map[string]string `json:"scripts"`
			}
			data, err := os.ReadFile(filepath.Join(root, "package.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &packageJSON); err != nil {
				t.Fatalf("package.json is not valid JSON: %v", err)
			}
			if packageJSON.Scripts["build"] == "" {
				t.Fatal("package.json must define a build script")
			}

			vite := readScaffoldFile(t, root, "vite.config.js")
			wantEntry := "resources/js/app.js"
			if frontend == FrontendInertiaReact {
				wantEntry = "resources/js/app.tsx"
			}
			if !strings.Contains(vite, wantEntry) {
				t.Fatalf("vite config does not include %s", wantEntry)
			}
		})
	}
}

func scaffoldContractProject(t *testing.T, cfg ProjectConfig) string {
	t.Helper()
	root := t.TempDir()
	if err := ScaffoldProject(cfg, root); err != nil {
		t.Fatal(err)
	}
	// This is the module replacement performed by `lemmego new`, kept offline.
	renameModule(cfg.ModuleName, root)
	return root
}

func assertGeneratedGoFiles(t *testing.T, root string) {
	t.Helper()
	fset := token.NewFileSet()
	count := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		count++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("{{")) {
			return fmt.Errorf("%s contains unresolved template delimiters", path)
		}
		if bytes.Contains(data, []byte("github.com/lemmego/lemmego")) {
			return fmt.Errorf("%s contains the scaffold module placeholder", path)
		}
		if _, err := parser.ParseFile(fset, path, data, parser.AllErrors); err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}
		if _, err := format.Source(data); err != nil {
			return fmt.Errorf("formatting %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("generated project contains no Go files")
	}
}

func assertGeneratedModule(t *testing.T, root, moduleName string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "module "+moduleName) {
		t.Fatalf("go.mod does not declare module %s", moduleName)
	}
	mainData, err := os.ReadFile(filepath.Join(root, "cmd", "app", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(mainData, []byte("app.WithConfig(config.GetAll())")) {
		t.Fatal("generated app must pass global configuration to app.Configure")
	}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(content, []byte("github.com/lemmego/lemmego")) {
			return fmt.Errorf("%s contains the scaffold module placeholder", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// MySQL refuses a UNIQUE index on a TEXT/BLOB column without a key length
// ("Error 1170 ... used in key specification without a key length"), so a
// scaffolded project that indexes one cannot migrate there at all. Every
// overlay had this on the users table, which made MySQL unusable for new
// projects whichever ORM was chosen.
func TestScaffoldMigrationsDoNotIndexTextColumns(t *testing.T) {
	migrations, err := filepath.Glob(filepath.Join("_scaffold", "overlays", "*", "internal", "migrations", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 {
		t.Fatal("no overlay migrations found; the glob is wrong")
	}

	// t.Text("x").Unique() and t.Text("x").Primary() are both unindexable.
	indexedText := regexp.MustCompile(`\.Text\((?:[^)]*)\)\.(Unique|Primary)\(\)`)

	for _, path := range migrations {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if match := indexedText.Find(source); match != nil {
			t.Errorf("%s indexes a TEXT column (%s); use String(name, length) so MySQL can index it",
				path, match)
		}
	}
}

// These assert on a generated project rather than on the template that
// produced it. Once a value sits behind a conditional, grepping the template
// stops proving that any project actually gets it — the string is present
// whether or not it is ever emitted.

// scaffoldForContract generates a project and returns its root.
func scaffoldForContract(t *testing.T, cfg ProjectConfig) string {
	t.Helper()
	dir := t.TempDir()
	if err := ScaffoldProject(cfg, dir); err != nil {
		t.Fatalf("ScaffoldProject() error = %v", err)
	}
	return dir
}

func readGenerated(t *testing.T, dir, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(body)
}

// With sessions disabled, an empty JWT secret meant auth.Check() could verify
// nothing and returned success: the protected area admitted anonymous visitors
// and the login page redirected everyone away. The provider must fall back to
// APP_KEY, which `lemmego new` always populates.
func TestScaffoldAuthHasAJwtSecretFallback(t *testing.T) {
	dir := scaffoldForContract(t, ProjectConfig{
		ModuleName: "example.com/auth", Preset: PresetRESTAPI,
		SQLDriver: SQLSQLite, EnableAuth: true,
	})

	providers := readGenerated(t, dir, "bootstrap/providers.go")
	if !strings.Contains(providers, `config.MustEnv("JWT_SECRET", config.MustEnv("APP_KEY", ""))`) {
		t.Error("the auth provider must fall back to APP_KEY; an empty secret leaves protected routes open")
	}
	if strings.Contains(providers, `config.MustEnv("JWT_SECRET", "")`) {
		t.Error("JWT_SECRET must not default to an empty string")
	}
}

// The app reads DB_CONNECTION but the migrate command reads DB_DRIVER, so a
// missing DB_DRIVER makes `lemmego run migrate up` fail on a fresh project
// with "driver is required".
func TestScaffoldEnvSetsDBDriver(t *testing.T) {
	for _, driver := range []SQLDriver{SQLSQLite, SQLMySQL, SQLPostgres} {
		t.Run(string(driver), func(t *testing.T) {
			dir := scaffoldForContract(t, ProjectConfig{
				ModuleName: "example.com/db", Preset: PresetRESTAPI, SQLDriver: driver,
			})

			env := readGenerated(t, dir, ".env.example")
			if !regexp.MustCompile(`(?m)^DB_DRIVER=` + string(driver) + `$`).MatchString(env) {
				t.Errorf("DB_DRIVER=%s must be set, or migrations fail out of the box:\n%s", driver, env)
			}
			if !regexp.MustCompile(`(?m)^DB_CONNECTION=` + driver.ConnectionName() + `$`).MatchString(env) {
				t.Errorf("DB_CONNECTION=%s is missing:\n%s", driver.ConnectionName(), env)
			}
		})
	}
}

// A project with no database must carry no database settings at all, rather
// than settings pointing at something that is not there.
func TestScaffoldEnvOmitsDatabaseSettingsWithoutADatabase(t *testing.T) {
	dir := scaffoldForContract(t, ProjectConfig{
		ModuleName: "example.com/nodb", Preset: PresetRESTAPI, SQLDriver: SQLNone,
		CacheDriver: CacheNone, QueueDriver: QueueNone,
	})

	env := readGenerated(t, dir, ".env.example")
	if regexp.MustCompile(`(?m)^DB_`).MatchString(env) {
		t.Errorf("a project without a database carries DB_ settings:\n%s", env)
	}
}

// MustEnv falls back only when a variable is absent, so a present-but-empty
// JWT_SECRET defeats the APP_KEY fallback and leaves auth unable to verify
// anything.
func TestScaffoldEnvDoesNotBlankJWTSecret(t *testing.T) {
	dir := scaffoldForContract(t, ProjectConfig{
		ModuleName: "example.com/jwt", Preset: PresetRESTAPI, SQLDriver: SQLSQLite, EnableAuth: true,
	})

	env := readGenerated(t, dir, ".env.example")
	if regexp.MustCompile(`(?m)^JWT_SECRET=\s*$`).MatchString(env) {
		t.Error("JWT_SECRET must be absent or populated, never set to an empty value")
	}
}

// The template cache is populated explicitly, so something has to call
// LoadTemplates. Nothing did, which left every .gohtml render failing with
// "template ... not found in cache" — the go_templates home page and the error
// pages of every preset.
func TestScaffoldLoadsTemplatesAtStartup(t *testing.T) {
	dir := scaffoldForContract(t, ProjectConfig{
		ModuleName: "example.com/tmpl", Preset: PresetMVC, Frontend: FrontendGoTemplates,
	})

	if !strings.Contains(readGenerated(t, dir, "cmd/app/main.go"), "res.LoadTemplates") {
		t.Error("main must load the template cache, or every .gohtml render fails")
	}
}

// A browser sends "text/html,...,*/*;q=0.8". The */* in that matches a JSON
// check, so testing for JSON first served browsers a raw JSON body instead of
// the error page.
func TestScaffoldErrorPagesPreferHTML(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("_scaffold", "base", "bootstrap", "errmap.go"))
	if err != nil {
		t.Fatal(err)
	}
	html := strings.Index(string(source), "c.WantsHTML()")
	json := strings.Index(string(source), "c.WantsJSON()")
	if html < 0 {
		t.Fatal("the error handler must check WantsHTML")
	}
	if json >= 0 && json < html {
		t.Error("WantsHTML must be checked before WantsJSON, or browsers receive JSON")
	}
}

// templ generate writes the _templ.go files that make the templates directory
// a Go package. Tidying before that runs leaves the project unable to resolve
// its own import of it.
func TestTemplGenerateRunsBeforeTidy(t *testing.T) {
	source, err := os.ReadFile("new.go")
	if err != nil {
		t.Fatal(err)
	}
	generate := strings.Index(string(source), `RunCommand(dirPath, "templ", "generate")`)
	tidy := strings.Index(string(source), "installGoModules(dirPath)")
	if generate < 0 || tidy < 0 {
		t.Fatal("expected both the templ generate and the module install steps")
	}
	if generate > tidy {
		t.Error("templ generate must run before installGoModules")
	}
}
