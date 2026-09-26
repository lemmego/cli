package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withFlags sets the `new` flags for one test and restores them after. They
// are one struct so that adding a flag does not mean editing this.
func withFlags(t *testing.T, flags projectFlagSet) {
	t.Helper()
	previous := projectFlags
	t.Cleanup(func() { projectFlags = previous })
	projectFlags = flags
}

func TestCollectNonInteractiveProjectConfig(t *testing.T) {
	withFlags(t, projectFlagSet{
		Module:   "github.com/example/app",
		Preset:   "mvc",
		ORM:      "bun",
		Database: "mysql",
		Frontend: "templ",
		Cache:    "redis",
		Queue:    "redis",
		Session:  "redis",
		Disk:     "s3",
		Auth:     true,
		GPA:      true,
	})

	cfg, err := collectNonInteractiveProjectConfig("app")
	if err != nil {
		t.Fatalf("collectNonInteractiveProjectConfig() error = %v", err)
	}
	if cfg.ModuleName != "github.com/example/app" || cfg.Preset != PresetMVC ||
		cfg.ORM != OrmBun || cfg.Frontend != FrontendTempl {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.SQLDriver != SQLMySQL || cfg.CacheDriver != CacheRedis ||
		cfg.QueueDriver != QueueRedis || cfg.SessionDriver != SessionRedis ||
		cfg.FilesystemDisk != DiskS3 {
		t.Fatalf("drivers not carried through: %+v", cfg)
	}
	if !cfg.EnableAuth || !cfg.EnableGPA {
		t.Fatalf("expected optional features enabled: %+v", cfg)
	}
}

func TestCollectNonInteractiveProjectConfigDefaultsAndValidation(t *testing.T) {
	withFlags(t, projectFlagSet{Module: "github.com/example/app"})

	cfg, err := collectNonInteractiveProjectConfig("app")
	if err != nil {
		t.Fatalf("collectNonInteractiveProjectConfig() error = %v", err)
	}
	if cfg.Preset != PresetMVC || cfg.ORM != OrmLemmego || cfg.Frontend != FrontendGoTemplates {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	// Every supported value is accepted, and nothing else is.
	for _, choice := range []OrmChoice{OrmLemmego, OrmGORM, OrmBun} {
		withFlags(t, projectFlagSet{Module: "github.com/example/app", ORM: string(choice)})
		cfg, err := collectNonInteractiveProjectConfig("app")
		if err != nil {
			t.Fatalf("--orm %s was rejected: %v", choice, err)
		}
		if cfg.ORM != choice {
			t.Fatalf("--orm %s resolved to %s", choice, cfg.ORM)
		}
	}

	withFlags(t, projectFlagSet{Module: "github.com/example/app", ORM: "prisma"})
	if _, err := collectNonInteractiveProjectConfig("app"); err == nil {
		t.Fatal("expected an unknown --orm to be rejected")
	}

	withFlags(t, projectFlagSet{})
	if _, err := collectNonInteractiveProjectConfig("app"); err == nil {
		t.Fatal("expected missing module error")
	}
}

// Each ORM choice must resolve to an auth overlay that actually exists, in
// both its plain and GPA forms.
func TestAuthOverlayExistsForEveryORM(t *testing.T) {
	for _, choice := range []OrmChoice{OrmLemmego, OrmGORM, OrmBun} {
		for _, gpa := range []bool{false, true} {
			cfg := ProjectConfig{Preset: PresetMVC, Frontend: FrontendGoTemplates, ORM: choice, EnableAuth: true, EnableGPA: gpa}
			overlays := resolveOverlays(cfg)

			var authOverlay string
			for _, overlay := range overlays {
				if strings.HasPrefix(overlay, "overlays/auth_") {
					authOverlay = overlay
				}
			}
			if authOverlay == "" {
				t.Fatalf("no auth overlay resolved for %s (gpa=%v)", choice, gpa)
			}
			if _, err := os.Stat(filepath.Join("_scaffold", authOverlay)); err != nil {
				t.Fatalf("%s resolves to %s, which does not exist: %v", choice, authOverlay, err)
			}
		}
	}
}

func TestFrontendPresetHasInertia(t *testing.T) {
	tests := []struct {
		preset FrontendPreset
		want   bool
	}{
		{FrontendGoTemplates, false},
		{FrontendTempl, false},
		{FrontendInertiaReact, true},
		{FrontendInertiaVue, true},
		{FrontendTemplInertiaReact, true},
		{FrontendTemplInertiaVue, true},
	}
	for _, tt := range tests {
		got := tt.preset.HasInertia()
		if got != tt.want {
			t.Errorf("%s.HasInertia() = %v, want %v", tt.preset, got, tt.want)
		}
	}
}

func TestFrontendPresetHasTempl(t *testing.T) {
	tests := []struct {
		preset FrontendPreset
		want   bool
	}{
		{FrontendGoTemplates, false},
		{FrontendTempl, true},
		{FrontendInertiaReact, false},
		{FrontendInertiaVue, false},
		{FrontendTemplInertiaReact, true},
		{FrontendTemplInertiaVue, true},
	}
	for _, tt := range tests {
		got := tt.preset.HasTempl()
		if got != tt.want {
			t.Errorf("%s.HasTempl() = %v, want %v", tt.preset, got, tt.want)
		}
	}
}

func TestFrontendPresetHasNodeDeps(t *testing.T) {
	tests := []struct {
		preset FrontendPreset
		want   bool
	}{
		{FrontendGoTemplates, false},
		{FrontendTempl, false},
		{FrontendInertiaReact, true},
		{FrontendInertiaVue, true},
		{FrontendTemplInertiaReact, true},
		{FrontendTemplInertiaVue, true},
	}
	for _, tt := range tests {
		got := tt.preset.HasNodeDeps()
		if got != tt.want {
			t.Errorf("%s.HasNodeDeps() = %v, want %v", tt.preset, got, tt.want)
		}
	}
}
