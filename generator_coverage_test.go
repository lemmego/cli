package cli

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
}

func readGeneratedFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(text, "<no value>") || strings.Contains(text, "{{") {
		t.Fatalf("generated file contains an unresolved template: %s", text)
	}
	return text
}

func assertGoSource(t *testing.T, path string) string {
	t.Helper()
	text := readGeneratedFile(t, path)
	if _, err := parser.ParseFile(token.NewFileSet(), path, text, parser.AllErrors); err != nil {
		t.Fatalf("generated Go source does not parse: %v\n%s", err, text)
	}
	return text
}

func TestGeneratorsWriteConfiguredPaths(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	seedPaths(t, projectPaths{
		HandlerPath:   "generated/handlers",
		InputPath:     "generated/inputs",
		ModelPath:     "generated/models",
		MigrationPath: "generated/migrations",
	})

	if err := NewHandlerGenerator(&HandlerConfig{Name: "user"}).Generate([]byte("// handler append\n")); err != nil {
		t.Fatal(err)
	}
	if got := assertGoSource(t, "generated/handlers/user_handlers.go"); !strings.Contains(got, "UserIndexHandler") || !strings.Contains(got, "handler append") {
		t.Fatalf("handler output missing expected content: %s", got)
	}

	if err := NewModelGenerator(&ModelConfig{Name: "user", Fields: []*ModelField{
		{Name: "id", Type: "uint64", Primary: true},
		{Name: "display_name", Type: "string", Required: true},
	}}).Generate(); err != nil {
		t.Fatal(err)
	}
	if got := assertGoSource(t, "generated/models/user.go"); !strings.Contains(got, "DisplayName string") {
		t.Fatalf("model output missing field: %s", got)
	}

	if err := NewInputGenerator(&InputConfig{Name: "user", Fields: []*InputField{
		{Name: "email", Type: "string", Required: true, Unique: true, Table: "users"},
		{Name: "avatar", Type: "file"},
	}}).Generate(); err != nil {
		t.Fatal(err)
	}
	if got := assertGoSource(t, "generated/inputs/user_input.go"); !strings.Contains(got, "multipart.FileHeader") || !strings.Contains(got, "api/db") {
		t.Fatalf("input output missing conditional imports: %s", got)
	}

	if err := NewMigrationGenerator(&MigrationConfig{
		TableName:      "users",
		Fields:         []*MigrationField{{Name: "email", Type: "string", Unique: true}, {Name: "team_id", Type: "unsignedBigInt", ForeignConstrained: true}},
		PrimaryColumns: []string{"email"},
		UniqueColumns:  [][]string{{"email"}},
		ForeignColumns: [][]string{{"team_id"}},
		Timestamps:     true,
	}).BumpVersion().Generate([]byte("// migration append\n")); err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob("generated/migrations/*.go")
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected one generated migration, got %v: %v", matches, err)
	}
	if got := assertGoSource(t, matches[0]); !strings.Contains(got, "created_at") || !strings.Contains(got, "migration append") {
		t.Fatalf("migration output missing expected content: %s", got)
	}
}

func TestFormGeneratorsWriteBothFlavors(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	fields := []*FormField{
		{Name: "email", Type: "text"},
		{Name: "remember_me", Type: "boolean"},
		{Name: "role", Type: "dropdown", Choices: []string{"Admin", "Member"}},
	}

	if err := NewFormGenerator(&FormConfig{Name: "sign_in", Flavor: "react", Route: "/login", Fields: fields}).Generate(); err != nil {
		t.Fatal(err)
	}
	react := readGeneratedFile(t, "resources/js/Pages/Forms/SignIn.tsx")
	if !strings.Contains(react, "useForm<SignInFields>") || !strings.Contains(react, "remember_me") {
		t.Fatalf("React form output missing expected content: %s", react)
	}

	if err := NewFormGenerator(&FormConfig{Name: "sign_in", Flavor: "templ", Route: "/login", Fields: fields}).Generate(); err != nil {
		t.Fatal(err)
	}
	templ := readGeneratedFile(t, "templates/sign_in.templ")
	if !strings.Contains(templ, "templ SignIn") || !strings.Contains(templ, "action=\"/login\"") {
		t.Fatalf("Templ form output missing expected content: %s", templ)
	}
}

func TestFormGeneratorRejectsUnknownFlavor(t *testing.T) {
	if err := NewFormGenerator(&FormConfig{Name: "user", Flavor: "svelte"}).Generate(); err == nil {
		t.Fatal("expected unknown form flavor to fail")
	}
}

func TestGeneratorHelpers(t *testing.T) {
	if got := NewDBTagBuilder(nil, "orm").Add("primaryKey", "").Add("index", "email").Build(); got != `orm:"primaryKey,index:email"` {
		t.Fatalf("DBTagBuilder.Build() = %q", got)
	}
	if got := NewDBTagBuilder(nil, "orm").Build(); got != "" {
		t.Fatalf("empty DBTagBuilder.Build() = %q", got)
	}

	if got := guessPluralizedTableNameFromColumnName("user_id"); got != "users" {
		t.Fatalf("guessPluralizedTableNameFromColumnName() = %q", got)
	}
	if got := guessPluralizedTableNameFromColumnName("status"); got != "statuses" {
		t.Fatalf("guessPluralizedTableNameFromColumnName() = %q", got)
	}

	migration := NewMigrationGenerator(&MigrationConfig{TableName: "users", Fields: []*MigrationField{}, Timestamps: true})
	if len(migration.fields) != 3 {
		t.Fatalf("timestamp fields = %d, want 3", len(migration.fields))
	}
	if got := migration.BumpVersion(); got != migration {
		t.Fatal("BumpVersion should return the same generator")
	}
}

func TestParseTemplateReturnsParseErrors(t *testing.T) {
	if _, err := ParseTemplate(nil, "{{", nil); err == nil {
		t.Fatal("expected malformed template to return an error")
	}
}

func TestGeneratorCommandsUseNonInteractiveArguments(t *testing.T) {
	root := t.TempDir()
	chdirForTest(t, root)
	seedPaths(t, projectPaths{
		HandlerPath:   "internal/handlers",
		InputPath:     "internal/inputs",
		ModelPath:     "internal/models",
		MigrationPath: "internal/migrations",
	})
	previousInteractive := shouldRunInteractively
	previousFlavor := flavor
	t.Cleanup(func() {
		shouldRunInteractively = previousInteractive
		flavor = previousFlavor
	})
	shouldRunInteractively = false
	flavor = "react"

	handlerCmd.Run(handlerCmd, []string{"post"})
	modelCmd.Run(modelCmd, []string{"post"})
	inputCmd.Run(inputCmd, []string{"post"})
	migrationCmd.Run(migrationCmd, []string{"posts"})
	formCmd.Run(formCmd, []string{"post_form"})
	if NewHandlerGenerator(&HandlerConfig{}).Command() == nil ||
		NewModelGenerator(&ModelConfig{}).Command() == nil ||
		NewInputGenerator(&InputConfig{}).Command() == nil ||
		NewMigrationGenerator(&MigrationConfig{}).Command() == nil {
		t.Fatal("generator returned a nil command")
	}
	handlerCmd.Run(handlerCmd, nil)
	modelCmd.Run(modelCmd, nil)
	inputCmd.Run(inputCmd, nil)
	migrationCmd.Run(migrationCmd, nil)
	formCmd.Run(formCmd, nil)

	for _, pattern := range []string{
		"internal/handlers/post_handlers.go",
		"internal/models/post.go",
		"internal/inputs/post_input.go",
		"internal/migrations/*.go",
		"resources/js/Pages/Forms/PostForm.tsx",
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			t.Errorf("command did not generate %s: %v", pattern, err)
		}
	}
}
