package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeFrontendPackages(t *testing.T) {
	packages, err := decodeFrontendPackages([]byte(`{"ImportPath":"example/app","Dir":"/tmp/app"}
{"ImportPath":"example/dep","Dir":"/tmp/dep","DepOnly":true}
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 2 || packages[1].DepOnly != true {
		t.Fatalf("decoded packages = %#v", packages)
	}
}

func TestBuildFrontendManifestDiscoversAndSortsPackages(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zeta", "alpha"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(dir, "frontend"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "frontend", "Page.tsx"), []byte("export default function Page() {}"), 0o644); err != nil {
			t.Fatal(err)
		}
		manifest := `{"schema":1,"id":"` + name + `","framework":"react","pages":{"` + name + `/Page":"frontend/Page.tsx"}}`
		if err := os.WriteFile(filepath.Join(dir, frontendManifestName), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	result, err := buildFrontendManifest([]frontendPackage{
		{ImportPath: "example/zeta", Dir: filepath.Join(root, "zeta")},
		{ImportPath: "example/alpha", Dir: filepath.Join(root, "alpha")},
		{ImportPath: "std", Dir: filepath.Join(root, "missing"), Standard: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Modules) != 2 || result.Modules[0].ID != "alpha" || result.Modules[1].ID != "zeta" {
		t.Fatalf("modules = %#v", result.Modules)
	}
	if result.Pages["alpha/Page"] != frontendSourcePath(filepath.Join(root, "alpha", "frontend", "Page.tsx")) {
		t.Fatalf("pages = %#v", result.Pages)
	}
}

func TestBuildFrontendManifestRejectsPageCollision(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"one", "two"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Join(dir, "frontend"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "frontend", "Page.tsx"), []byte("page"), 0o644); err != nil {
			t.Fatal(err)
		}
		manifest := `{"schema":1,"id":"` + name + `","framework":"react","pages":{"Shared":"frontend/Page.tsx"}}`
		if err := os.WriteFile(filepath.Join(dir, frontendManifestName), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := buildFrontendManifest([]frontendPackage{
		{ImportPath: "example/one", Dir: filepath.Join(root, "one")},
		{ImportPath: "example/two", Dir: filepath.Join(root, "two")},
	})
	if err == nil {
		t.Fatal("expected page collision")
	}
}

func TestValidateFrontendPathRejectsEscape(t *testing.T) {
	if err := validateFrontendPath("/tmp/package", "../outside.tsx"); err == nil {
		t.Fatal("expected path escape to be rejected")
	}
}
