package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type templateData struct {
	// ProjectConfig is embedded, so every choice and predicate on it is
	// reachable from a template as {{.CacheDriver}} or {{.HasDatabase}}.
	//
	// Nothing here may shadow one of its fields. SessionDriver used to be
	// declared here as a string, which shadowed the embedded choice: the
	// template read one value while the configuration held another.
	ProjectConfig

	versions         map[string]string
	InertiaProvider  bool
	FrontendHasTempl bool
	FrontendHasReact bool
	FrontendHasVue   bool
	HasTempl         bool
}

func (td templateData) Version(pkg string) string {
	if td.versions != nil {
		if v, ok := td.versions[pkg]; ok {
			return v
		}
	}
	return "latest"
}

// scaffoldSource bundles an fs.FS with its internal path prefix.
type scaffoldSource struct {
	fs     fs.FS
	prefix string
}

// resolveScaffoldSource returns the scaffold source, preferring the cached
// scaffold on disk over the embedded one.
func resolveScaffoldSource() scaffoldSource {
	if embeddedScaffoldOnly() {
		return scaffoldSource{fs: scaffoldEmbedFS, prefix: "_scaffold"}
	}
	if dir := scaffoldDir(); dir != "" {
		return scaffoldSource{fs: os.DirFS(dir), prefix: ""}
	}
	return scaffoldSource{fs: scaffoldEmbedFS, prefix: "_scaffold"}
}

// loadVersions reads versions.json from the scaffold source.
// Returns nil if unavailable (offline), so Version() falls back to "latest".
func loadVersions(src scaffoldSource) map[string]string {
	p := path.Join(src.prefix, "stubs", "versions.json")
	data, err := fs.ReadFile(src.fs, p)
	if err != nil {
		return nil
	}
	var versions map[string]string
	if err := json.Unmarshal(data, &versions); err != nil {
		return nil
	}
	return versions
}

func ScaffoldProject(cfg ProjectConfig, destDir string) error {
	// Fill in whatever the caller left unset, so a partially populated
	// configuration scaffolds the same project as a fully specified one.
	normalize(&cfg)
	if err := validate(cfg); err != nil {
		return err
	}

	if !embeddedScaffoldOnly() {
		fetchLatestScaffold()
	}
	src := resolveScaffoldSource()

	td := buildTemplateData(cfg)
	td.versions = loadVersions(src)
	overlays := resolveOverlays(cfg)

	fmt.Println("> Scaffolding project...")

	if err := copyBaseFiles(destDir, src); err != nil {
		return fmt.Errorf("copying base files: %w", err)
	}

	ensureDirs(destDir,
		"internal/commands",
		"internal/handlers",
		"internal/middleware",
		"internal/models",
		"internal/plugins",
		"public",
		"storage",
		"storage/session",
	)

	for _, overlay := range overlays {
		if err := applyOverlay(overlay, destDir, src); err != nil {
			return fmt.Errorf("applying overlay %s: %w", overlay, err)
		}
	}

	if err := generateDynamicFiles(td, destDir, src); err != nil {
		return fmt.Errorf("generating dynamic files: %w", err)
	}

	if err := formatGoFiles(destDir); err != nil {
		return fmt.Errorf("formatting generated code: %w", err)
	}

	return nil
}

// formatGoFiles gofmts the generated tree.
//
// Scaffold sources are written against the template module path and rewritten
// to the project's own, which reorders imports and leaves the result
// unformatted. Formatting afterwards is the only place that can know the final
// paths, and DEVELOPMENT.md requires generated projects to pass formatting.
//
// A file that fails to parse is left as it is rather than failing the whole
// scaffold: a readable broken file is easier to diagnose than an aborted
// generation.
func formatGoFiles(destDir string) error {
	return filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := format.Source(source)
		if err != nil {
			return nil
		}
		if bytes.Equal(source, formatted) {
			return nil
		}
		return os.WriteFile(path, formatted, info.Mode().Perm())
	})
}

func buildTemplateData(cfg ProjectConfig) templateData {
	normalize(&cfg)

	return templateData{
		ProjectConfig:    cfg,
		InertiaProvider:  cfg.Frontend.HasInertia(),
		HasTempl:         cfg.Frontend.HasTempl() || cfg.Preset == PresetRESTAPI,
		FrontendHasTempl: cfg.Frontend.HasTempl(),
		FrontendHasReact: cfg.Frontend == FrontendInertiaReact || cfg.Frontend == FrontendTemplInertiaReact,
		FrontendHasVue:   cfg.Frontend == FrontendInertiaVue || cfg.Frontend == FrontendTemplInertiaVue,
	}
}

func resolveOverlays(cfg ProjectConfig) []string {
	// Normalize here too rather than relying on the caller. A configuration
	// with unset fields would otherwise resolve different overlays than the
	// same configuration after ScaffoldProject filled it in.
	normalize(&cfg)

	var overlays []string

	if cfg.Preset == PresetMVC {
		overlays = append(overlays, "overlays/mvc")

		switch cfg.Frontend {
		case FrontendGoTemplates:
			overlays = append(overlays, "overlays/frontend_go_templates")
		case FrontendTempl:
			overlays = append(overlays, "overlays/frontend_templ")
		case FrontendInertiaReact:
			overlays = append(overlays, "overlays/frontend_inertia_react")
		case FrontendInertiaVue:
			overlays = append(overlays, "overlays/frontend_inertia_vue")
		case FrontendTemplInertiaReact:
			overlays = append(overlays, "overlays/frontend_templ_inertia_react")
		case FrontendTemplInertiaVue:
			overlays = append(overlays, "overlays/frontend_templ_inertia_vue")
		}
	} else {
		overlays = append(overlays, "overlays/rest_api")
	}

	// The auth overlay scaffolds a model, repositories and a users-table
	// migration that all resolve a database connection, so it follows the
	// database rather than the auth flag alone. validate rejects that
	// combination before reaching here; this is the second line of defence,
	// and without it the overlay name would come out as "overlays/auth_none".
	if cfg.EnableAuth && cfg.HasDatabase() {
		overlay := "overlays/auth_" + string(cfg.ORM)
		if cfg.EnableGPA {
			overlay += "_gpa"
		}
		overlays = append(overlays, overlay)
	}

	return overlays
}

func copyBaseFiles(destDir string, src scaffoldSource) error {
	root := path.Join(src.prefix, "base")

	return fs.WalkDir(src.fs, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(p, root+"/")
		if relPath == root || relPath == "" {
			return nil
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(relPath))

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := fs.ReadFile(src.fs, p)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func applyOverlay(overlayPath string, destDir string, src scaffoldSource) error {
	root := path.Join(src.prefix, overlayPath)

	// An overlay that was named but is not in the scaffold means either a
	// naming bug — the auth overlay path is assembled from the SQL layer — or
	// a cached scaffold from a different CLI. Ignoring it produced a project
	// missing its models, repositories and migrations that still scaffolded,
	// formatted, and reported success; the failure surfaced much later as
	// compile errors pointing somewhere else.
	if _, err := fs.Stat(src.fs, root); err != nil {
		return fmt.Errorf("overlay %s is not present in this scaffold: %w", overlayPath, err)
	}

	return fs.WalkDir(src.fs, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(p, root+"/")
		if relPath == root || relPath == "" {
			return nil
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(relPath))

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := fs.ReadFile(src.fs, p)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func generateDynamicFiles(td templateData, destDir string, src scaffoldSource) error {
	// A subsystem the project left out contributes no configuration file. The
	// file is omitted by not naming it here, which is how the frontend and
	// node stubs already work.
	stubs := map[string]string{
		"bootstrap/providers.go":          "providers.go.tpl",
		"bootstrap/routes.go":             "routes_bootstrap.go.tpl",
		"bootstrap/middleware.go":         "middleware.go.tpl",
		"internal/configs/session.go":     "session.go.tpl",
		"internal/configs/filesystems.go": "filesystems.go.tpl",
		"internal/routes/api.go":          "api.go.tpl",
		"cmd/app/main.go":                 "main.go.tpl",
		"go.mod":                          "go.mod.tpl",
		".env.example":                    "env.example.tpl",
	}

	if td.Preset == PresetMVC {
		stubs["internal/routes/web.go"] = "web.go.tpl"
	}
	if td.HasDatabase() {
		stubs["internal/configs/database.go"] = "database.go.tpl"
	}
	if td.HasCache() {
		stubs["internal/configs/cache.go"] = "cache.go.tpl"
	}
	if td.HasQueue() {
		stubs["internal/configs/tasker.go"] = "tasker.go.tpl"
	}
	// One shared Redis connection, written when anything needs one. It lives
	// in its own file rather than inside the database config, which a project
	// without a database does not have.
	if td.UsesRedis() {
		stubs["internal/configs/keyvalue.go"] = "keyvalue.go.tpl"
	}

	if td.Frontend.HasNodeDeps() {
		stubs["pnpm-workspace.yaml"] = "pnpm-workspace.yaml.tpl"
		stubs["package.json"] = "package.json.tpl"
		stubs["tsconfig.json"] = "tsconfig.json.tpl"
		stubs["vite.config.js"] = "vite.config.js.tpl"
	}

	stubsDir := path.Join(src.prefix, "stubs")

	for destRelPath, stubRelPath := range stubs {
		stubFullPath := path.Join(stubsDir, stubRelPath)
		tmplData, err := fs.ReadFile(src.fs, stubFullPath)
		if err != nil {
			return fmt.Errorf("reading stub %s: %w", stubFullPath, err)
		}

		tmpl, err := template.New(stubRelPath).Parse(string(tmplData))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", stubRelPath, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, td); err != nil {
			return fmt.Errorf("executing template %s: %w", stubRelPath, err)
		}

		destPath := filepath.Join(destDir, filepath.FromSlash(destRelPath))
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(destPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", destRelPath, err)
		}
	}

	return nil
}

func hasNodeDeps(cfg ProjectConfig) bool {
	return cfg.Preset == PresetMVC && cfg.Frontend.HasNodeDeps()
}

func hasTemplGenerate(cfg ProjectConfig) bool {
	return cfg.Preset == PresetMVC && cfg.Frontend.HasTempl()
}

func ensureDirs(destDir string, dirs ...string) {
	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(destDir, dir), 0755)
	}
}
