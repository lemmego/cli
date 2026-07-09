package cli

import "testing"

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
