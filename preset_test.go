package cli

import "testing"

func TestCollectNonInteractiveProjectConfig(t *testing.T) {
	oldModule, oldPreset, oldORM, oldFrontend := projectModule, projectPreset, projectORM, projectFrontend
	oldRedis, oldAuth, oldGPA := projectRedis, projectAuth, projectGPA
	t.Cleanup(func() {
		projectModule, projectPreset, projectORM, projectFrontend = oldModule, oldPreset, oldORM, oldFrontend
		projectRedis, projectAuth, projectGPA = oldRedis, oldAuth, oldGPA
	})

	projectModule = "github.com/example/app"
	projectPreset = "mvc"
	projectORM = "bun"
	projectFrontend = "templ"
	projectRedis = true
	projectAuth = true
	projectGPA = true

	cfg, err := collectNonInteractiveProjectConfig("app")
	if err != nil {
		t.Fatalf("collectNonInteractiveProjectConfig() error = %v", err)
	}
	if cfg.ModuleName != projectModule || cfg.Preset != PresetMVC || cfg.ORM != OrmBun || cfg.Frontend != FrontendTempl {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if !cfg.EnableRedis || !cfg.EnableAuth || !cfg.EnableGPA {
		t.Fatalf("expected optional features enabled: %+v", cfg)
	}
}

func TestCollectNonInteractiveProjectConfigDefaultsAndValidation(t *testing.T) {
	oldModule, oldPreset, oldORM, oldFrontend := projectModule, projectPreset, projectORM, projectFrontend
	oldRedis, oldAuth, oldGPA := projectRedis, projectAuth, projectGPA
	t.Cleanup(func() {
		projectModule, projectPreset, projectORM, projectFrontend = oldModule, oldPreset, oldORM, oldFrontend
		projectRedis, projectAuth, projectGPA = oldRedis, oldAuth, oldGPA
	})

	projectModule, projectPreset, projectORM, projectFrontend = "github.com/example/app", "", "", ""
	projectRedis, projectAuth, projectGPA = false, false, false
	cfg, err := collectNonInteractiveProjectConfig("app")
	if err != nil {
		t.Fatalf("collectNonInteractiveProjectConfig() error = %v", err)
	}
	if cfg.Preset != PresetMVC || cfg.ORM != OrmGORM || cfg.Frontend != FrontendGoTemplates {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	projectModule = ""
	if _, err := collectNonInteractiveProjectConfig("app"); err == nil {
		t.Fatal("expected missing module error")
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
