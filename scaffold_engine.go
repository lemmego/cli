package cli

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type templateData struct {
	ProjectConfig
	SessionDriver    string
	InertiaProvider  bool
	FrontendHasTempl bool
	FrontendHasReact bool
	FrontendHasVue   bool
	HasTempl         bool
}

func (td templateData) Version(pkg string) string {
	v, ok := DependencyVersions[pkg]
	if !ok {
		return "latest"
	}
	return v
}

func ScaffoldProject(cfg ProjectConfig, destDir string) error {
	td := buildTemplateData(cfg)
	overlays := resolveOverlays(cfg)

	fmt.Println("> Scaffolding project...")

	if err := copyBaseFiles(destDir); err != nil {
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
		if err := applyOverlay(overlay, destDir); err != nil {
			return fmt.Errorf("applying overlay %s: %w", overlay, err)
		}
	}

	if err := generateDynamicFiles(td, destDir); err != nil {
		return fmt.Errorf("generating dynamic files: %w", err)
	}

	return nil
}

func buildTemplateData(cfg ProjectConfig) templateData {
	td := templateData{
		ProjectConfig:   cfg,
		SessionDriver:   "file",
		InertiaProvider: cfg.Frontend.HasInertia(),
		HasTempl:        cfg.Frontend.HasTempl() || cfg.Preset == PresetRESTAPI,
		FrontendHasTempl: cfg.Frontend.HasTempl(),
		FrontendHasReact: cfg.Frontend == FrontendInertiaReact || cfg.Frontend == FrontendTemplInertiaReact,
		FrontendHasVue:   cfg.Frontend == FrontendInertiaVue || cfg.Frontend == FrontendTemplInertiaVue,
	}
	if cfg.EnableRedis {
		td.SessionDriver = "redis"
	}
	return td
}

func resolveOverlays(cfg ProjectConfig) []string {
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

	if cfg.EnableAuth {
		if cfg.ORM == OrmGORM {
			overlays = append(overlays, "overlays/auth_gorm")
		} else {
			overlays = append(overlays, "overlays/auth_bun")
		}
	}

	return overlays
}

func copyBaseFiles(destDir string) error {
	return fs.WalkDir(scaffoldFS, "_scaffold/base", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, "_scaffold/base/")
		if relPath == "_scaffold/base" {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := fs.ReadFile(scaffoldFS, path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func applyOverlay(overlayPath string, destDir string) error {
	fullPath := "_scaffold/" + overlayPath

	_, err := fs.Stat(scaffoldFS, fullPath)
	if err != nil {
		return nil
	}

	return fs.WalkDir(scaffoldFS, fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath := strings.TrimPrefix(path, fullPath+"/")
		if relPath == fullPath {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := fs.ReadFile(scaffoldFS, path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func generateDynamicFiles(td templateData, destDir string) error {
	stubs := map[string]string{
		"bootstrap/providers.go":  "_scaffold/stubs/providers.go.tpl",
		"bootstrap/routes.go":     "_scaffold/stubs/routes_bootstrap.go.tpl",
		"bootstrap/middleware.go":  "_scaffold/stubs/middleware.go.tpl",
		"internal/configs/database.go": "_scaffold/stubs/database.go.tpl",
		"internal/configs/session.go":  "_scaffold/stubs/session.go.tpl",
		"cmd/app/main.go":         "_scaffold/stubs/main.go.tpl",
		"go.mod":                  "_scaffold/stubs/go.mod.tpl",
		".env.example":            "_scaffold/stubs/env.example.tpl",
	}

	if td.Preset == PresetMVC {
		stubs["internal/routes/web.go"] = "_scaffold/stubs/web.go.tpl"
	}

	stubs["internal/routes/api.go"] = "_scaffold/stubs/api.go.tpl"

	if td.Frontend.HasNodeDeps() {
		stubs["package.json"] = "_scaffold/stubs/package.json.tpl"
		stubs["vite.config.js"] = "_scaffold/stubs/vite.config.js.tpl"
	}

	for destRelPath, stubPath := range stubs {
		tmplData, err := fs.ReadFile(scaffoldFS, stubPath)
		if err != nil {
			return fmt.Errorf("reading stub %s: %w", stubPath, err)
		}

		tmpl, err := template.New(stubPath).Parse(string(tmplData))
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", stubPath, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, td); err != nil {
			return fmt.Errorf("executing template %s: %w", stubPath, err)
		}

		destPath := filepath.Join(destDir, destRelPath)
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
