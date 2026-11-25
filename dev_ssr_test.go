package cli

import (
	"os"
	"strings"
	"testing"
)

func TestStandaloneRendererPortDefaultsAndCanBeConfigured(t *testing.T) {
	t.Setenv("INERTIA_SSR_PORT", "")
	port, err := standaloneRendererPort()
	if err != nil || port != 13714 {
		t.Fatalf("unset port = %d, %v; want 13714", port, err)
	}

	t.Setenv("INERTIA_SSR_PORT", "14714")
	port, err = standaloneRendererPort()
	if err != nil || port != 14714 {
		t.Fatalf("configured port = %d, %v; want 14714", port, err)
	}
}

func TestStandaloneRendererPortRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"abc", "0", "65536", "-1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("INERTIA_SSR_PORT", value)
			if _, err := standaloneRendererPort(); err == nil {
				t.Fatalf("port %q was accepted", value)
			}
		})
	}
}

func TestDevSSRCanBeDisabled(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"", false},
		{"true", false},
		{"1", false},
		{"false", true},
		{"0", true},
		{"f", true},
		{"invalid", false},
	} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("INERTIA_SSR", tc.value)
			if got := devSSRDisabled(); got != tc.want {
				t.Fatalf("devSSRDisabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInertiaSSRCheckUsesConfiguredPort(t *testing.T) {
	t.Setenv("INERTIA_SSR_PORT", "14714")
	if err := inertiaSSRCheckCmd.PreRunE(inertiaSSRCheckCmd, nil); err != nil {
		t.Fatal(err)
	}
	if ssrPort != 14714 {
		t.Fatalf("check port = %d, want 14714", ssrPort)
	}
}

func TestInertiaSSRTemplatesPassConfiguredPortAtRuntime(t *testing.T) {
	for _, path := range []string{
		"_scaffold/overlays/frontend_inertia_react/resources/js/ssr.tsx",
		"_scaffold/overlays/frontend_templ_inertia_react/resources/js/ssr.tsx",
		"_scaffold/overlays/frontend_inertia_vue/resources/js/ssr.js",
		"_scaffold/overlays/frontend_templ_inertia_vue/resources/js/ssr.js",
	} {
		t.Run(path, func(t *testing.T) {
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"createSSRServer", "INERTIA_SSR_PORT", "port: ssrPort"} {
				if !strings.Contains(string(contents), want) {
					t.Errorf("%s does not contain %q", path, want)
				}
			}
		})
	}
}
