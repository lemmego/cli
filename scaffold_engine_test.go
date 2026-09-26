package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Scaffold tests must always exercise the checked-in embedded fixtures.
	_ = os.Setenv("LEMMEGO_SCAFFOLD_SOURCE", "embedded")
	os.Exit(m.Run())
}

func TestBuildTemplateData(t *testing.T) {
	cfg := ProjectConfig{
		Name:       "testapp",
		ModuleName: "github.com/test/app",
		Preset:     PresetMVC,
		ORM:        OrmGORM,
		EnableAuth: true,
		Frontend:   FrontendInertiaReact,
	}
	td := buildTemplateData(cfg)

	if td.Name != "testapp" {
		t.Errorf("expected testapp, got %s", td.Name)
	}
	if !td.InertiaProvider {
		t.Error("expected InertiaProvider true for inertiareact frontend")
	}
	if !td.FrontendHasReact {
		t.Error("expected FrontendHasReact true")
	}
	if td.FrontendHasVue {
		t.Error("expected FrontendHasVue false")
	}
	if td.HasTempl {
		t.Error("expected HasTempl false for inertiareact")
	}
	if td.EnableAuth != true {
		t.Error("expected EnableAuth true")
	}
}

func TestBuildTemplateDataGoTemplates(t *testing.T) {
	cfg := ProjectConfig{
		Preset:   PresetMVC,
		ORM:      OrmGORM,
		Frontend: FrontendGoTemplates,
	}
	td := buildTemplateData(cfg)
	if td.InertiaProvider {
		t.Error("expected InertiaProvider false for go_templates")
	}
	if td.HasTempl {
		t.Error("expected HasTempl false for go_templates")
	}
}

func TestBuildTemplateDataTempl(t *testing.T) {
	cfg := ProjectConfig{
		Preset:   PresetMVC,
		ORM:      OrmGORM,
		Frontend: FrontendTempl,
	}
	td := buildTemplateData(cfg)
	if !td.HasTempl {
		t.Error("expected HasTempl true for templ frontend")
	}
	if td.InertiaProvider {
		t.Error("expected InertiaProvider false for templ only")
	}
}

func TestBuildTemplateDataTemplInertiaReact(t *testing.T) {
	cfg := ProjectConfig{
		Preset:   PresetMVC,
		ORM:      OrmGORM,
		Frontend: FrontendTemplInertiaReact,
	}
	td := buildTemplateData(cfg)
	if !td.HasTempl {
		t.Error("expected HasTempl true")
	}
	if !td.InertiaProvider {
		t.Error("expected InertiaProvider true")
	}
	if !td.FrontendHasReact {
		t.Error("expected FrontendHasReact true")
	}
}

func TestBuildTemplateDataRestAPI(t *testing.T) {
	cfg := ProjectConfig{
		Preset: PresetRESTAPI,
		ORM:    OrmGORM,
	}
	td := buildTemplateData(cfg)
	if td.InertiaProvider {
		t.Error("expected InertiaProvider false for rest_api")
	}
	if !td.HasTempl {
		t.Error("expected HasTempl true for rest_api")
	}
}

func TestBuildTemplateDataSessionDriver(t *testing.T) {
	cfg := ProjectConfig{
		Preset:      PresetMVC,
		ORM:         OrmGORM,
		EnableRedis: true,
		Frontend:    FrontendGoTemplates,
	}
	td := buildTemplateData(cfg)
	if td.SessionDriver != "redis" {
		t.Errorf("expected redis session driver, got %s", td.SessionDriver)
	}

	cfg2 := ProjectConfig{
		Preset:      PresetMVC,
		ORM:         OrmGORM,
		EnableRedis: false,
		Frontend:    FrontendGoTemplates,
	}
	td2 := buildTemplateData(cfg2)
	if td2.SessionDriver != "file" {
		t.Errorf("expected file session driver, got %s", td2.SessionDriver)
	}
}

func TestResolveOverlaysMVCDefault(t *testing.T) {
	cfg := ProjectConfig{
		Preset:      PresetMVC,
		ORM:         OrmGORM,
		Frontend:    FrontendGoTemplates,
		EnableRedis: false,
		EnableAuth:  false,
	}
	overlays := resolveOverlays(cfg)
	if len(overlays) == 0 {
		t.Fatal("expected at least one overlay")
	}
}

func TestResolveOverlaysWithAuth(t *testing.T) {
	cfg := ProjectConfig{
		Preset:     PresetMVC,
		ORM:        OrmGORM,
		Frontend:   FrontendGoTemplates,
		EnableAuth: true,
	}
	overlays := resolveOverlays(cfg)
	found := false
	for _, o := range overlays {
		if o == "overlays/auth_gorm" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected auth_gorm overlay, got %v", overlays)
	}
}

func TestResolveOverlaysWithAuthBunGPA(t *testing.T) {
	cfg := ProjectConfig{
		Preset:     PresetRESTAPI,
		ORM:        OrmBun,
		EnableAuth: true,
		EnableGPA:  true,
	}
	overlays := resolveOverlays(cfg)
	found := false
	for _, o := range overlays {
		if o == "overlays/auth_bun_gpa" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected auth_bun_gpa overlay, got %v", overlays)
	}
}

func TestScaffoldProjectCreatesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ProjectConfig{
		Name:       "testproj",
		ModuleName: "github.com/test/testproj",
		Preset:     PresetRESTAPI,
		ORM:        OrmGORM,
		Frontend:   FrontendGoTemplates,
	}

	err := ScaffoldProject(cfg, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expectedFiles := []string{
		"go.mod",
		"cmd/app/main.go",
		"bootstrap/providers.go",
		"bootstrap/commands.go",
		"bootstrap/routes.go",
		"internal/configs/app.go",
		"internal/configs/database.go",
		"internal/configs/session.go",
		"internal/configs/cache.go",
		"internal/commands/appkey.go",
		"internal/commands/inspire.go",
	}

	for _, f := range expectedFiles {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s was not created", f)
		}
	}
}

func TestScaffoldVueUsesVueEntryAndPage(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ProjectConfig{
		Name:       "vueapp",
		ModuleName: "github.com/test/vueapp",
		Preset:     PresetMVC,
		ORM:        OrmGORM,
		Frontend:   FrontendInertiaVue,
	}

	if err := ScaffoldProject(cfg, tmpDir); err != nil {
		t.Fatal(err)
	}

	root := readScaffoldFile(t, tmpDir, "resources/views/root.html")
	if !strings.Contains(root, `resources/js/app.js`) || strings.Contains(root, `resources/js/app.tsx`) {
		t.Fatalf("Vue root has incorrect entry: %s", root)
	}
	routes := readScaffoldFile(t, tmpDir, "internal/routes/web.go")
	if !strings.Contains(routes, `inertia.Respond(c, "IndexVue"`) || strings.Contains(routes, "IndexReact") {
		t.Fatalf("Vue route has incorrect page: %s", routes)
	}
}

func TestScaffoldNodePresetEmitsTSConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ProjectConfig{Preset: PresetMVC, ORM: OrmGORM, Frontend: FrontendInertiaVue}
	if err := ScaffoldProject(cfg, tmpDir); err != nil {
		t.Fatal(err)
	}
	if content := readScaffoldFile(t, tmpDir, "tsconfig.json"); !strings.Contains(content, `"jsx": "preserve"`) {
		t.Fatalf("expected Vue tsconfig, got: %s", content)
	}
}

func TestScaffoldAlwaysImportsMigrations(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ProjectConfig{Preset: PresetRESTAPI, ORM: OrmGORM}
	if err := ScaffoldProject(cfg, tmpDir); err != nil {
		t.Fatal(err)
	}
	main := readScaffoldFile(t, tmpDir, "cmd/app/main.go")
	if !strings.Contains(main, `_ "github.com/lemmego/lemmego/internal/migrations"`) {
		t.Fatal("expected migrations import for projects without auth")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "internal/migrations/migrations.go")); err != nil {
		t.Fatalf("expected base migrations package: %v", err)
	}
}

func TestScaffoldAuthReadsJWTSecretFromEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ProjectConfig{Preset: PresetRESTAPI, ORM: OrmGORM, EnableAuth: true}
	if err := ScaffoldProject(cfg, tmpDir); err != nil {
		t.Fatal(err)
	}
	providers := readScaffoldFile(t, tmpDir, "bootstrap/providers.go")
	if !strings.Contains(providers, `config.MustEnv("JWT_SECRET"`) || strings.Contains(providers, "a-long-long-secret") {
		t.Fatalf("JWT secret is not environment-backed: %s", providers)
	}
	// An empty fallback leaves auth unable to verify anything, which reads as
	// success: protected routes open up and the login page becomes
	// unreachable. APP_KEY is always populated by `lemmego new`.
	if !strings.Contains(providers, `config.MustEnv("APP_KEY", "")`) {
		t.Fatalf("the JWT secret must fall back to APP_KEY: %s", providers)
	}
}

func readScaffoldFile(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func TestScaffoldProjectModuleRenamed(t *testing.T) {
	tmpDir := t.TempDir()
	module := "github.com/mycompany/myapp"
	cfg := ProjectConfig{
		Name:       "myapp",
		ModuleName: module,
		Preset:     PresetRESTAPI,
		ORM:        OrmGORM,
		Frontend:   FrontendGoTemplates,
	}

	err := ScaffoldProject(cfg, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	gomodPath := filepath.Join(tmpDir, "go.mod")
	content, err := os.ReadFile(gomodPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != module+"\n" && !contains(string(content), module) {
		t.Errorf("expected go.mod to contain module %s", module)
	}
}

func TestScaffoldProjectSQLiteCreated(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := ProjectConfig{
		Name:       "sqliteapp",
		ModuleName: "github.com/test/sqliteapp",
		Preset:     PresetRESTAPI,
		ORM:        OrmGORM,
		Frontend:   FrontendGoTemplates,
	}

	ScaffoldProject(cfg, tmpDir)
	createSQLiteDatabase(tmpDir)

	dbPath := filepath.Join(tmpDir, "storage", "database.sqlite")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected storage/database.sqlite to exist")
	}
}

func TestVersionLookup(t *testing.T) {
	td := templateData{versions: map[string]string{
		"github.com/lemmego/api": "v0.1.26",
	}}
	v := td.Version("github.com/lemmego/api")
	if v != "v0.1.26" {
		t.Errorf("expected v0.1.26, got %s", v)
	}
	v = td.Version("nonexistent")
	if v != "latest" {
		t.Errorf("expected 'latest' for missing entry, got %s", v)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
