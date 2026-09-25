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

// With sessions disabled, an empty JWT secret meant auth.Check() could verify
// nothing and returned success: the protected area admitted anonymous visitors
// and the login page redirected everyone away. The stub must fall back to
// APP_KEY, which `lemmego new` always populates.
func TestScaffoldAuthHasAJwtSecretFallback(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("_scaffold", "stubs", "providers.go.tpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), `config.MustEnv("JWT_SECRET", config.MustEnv("APP_KEY", ""))`) {
		t.Error("the auth provider must fall back to APP_KEY; an empty secret leaves protected routes open")
	}
	if strings.Contains(string(source), `config.MustEnv("JWT_SECRET", "")`) {
		t.Error("JWT_SECRET must not default to an empty string")
	}
}

// The app reads DB_CONNECTION but the migrate command reads DB_DRIVER, so a
// commented-out DB_DRIVER makes `lemmego run migrate up` fail on a fresh
// project with "driver is required".
func TestScaffoldEnvSetsDBDriver(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("_scaffold", "stubs", "env.example.tpl"))
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`(?m)^DB_DRIVER=`).Match(source) {
		t.Error("DB_DRIVER must be set, not commented out, or migrations fail out of the box")
	}
}

// MustEnv falls back only when a variable is absent, so a present-but-empty
// JWT_SECRET defeats the APP_KEY fallback and leaves auth unable to verify
// anything.
func TestScaffoldEnvDoesNotBlankJWTSecret(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("_scaffold", "stubs", "env.example.tpl"))
	if err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`(?m)^JWT_SECRET=\s*$`).Match(source) {
		t.Error("JWT_SECRET must be absent or populated, never set to an empty value")
	}
}
