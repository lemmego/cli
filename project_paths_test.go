package cli

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/lemmego/fsys"
)

// Outside a project there is nothing to ask, so the conventional paths stand
// and no build is attempted.
func TestPathsOutsideAProject(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	paths := resolveProjectPaths()
	if paths != conventionalPaths() {
		t.Errorf("paths = %+v, want the conventional set", paths)
	}
}

// A project that answers only some of the question still produces a usable
// set, rather than writing files to "".
func TestPartialAnswersAreFilledIn(t *testing.T) {
	answered := projectPaths{ModelPath: "./domain/models"}

	merged := merge(answered, conventionalPaths())

	if merged.ModelPath != "./domain/models" {
		t.Errorf("ModelPath = %q, want the project's answer", merged.ModelPath)
	}
	if merged.HandlerPath != conventionalPaths().HandlerPath {
		t.Errorf("HandlerPath = %q, want the conventional path", merged.HandlerPath)
	}
	if merged.MigrationPath == "" {
		t.Error("an unanswered path was left empty")
	}
}

// The project logs during boot before printing its answer, so the JSON is the
// last line rather than the whole output.
func TestTheAnswerIsTakenFromTheLastLine(t *testing.T) {
	output := []byte("INFO: connecting to the database\nINFO: ready\n{\"model_path\":\"./x\"}\n")

	if got := string(lastLine(output)); got != `{"model_path":"./x"}` {
		t.Errorf("lastLine() = %q", got)
	}
}

func TestLastLineOfASingleLine(t *testing.T) {
	if got := string(lastLine([]byte(`{"model_path":"./x"}`))); got != `{"model_path":"./x"}` {
		t.Errorf("lastLine() = %q", got)
	}
}

// A project can put its code somewhere that does not exist yet, and should not
// have to create the directory by hand first.
func TestEnsurePackageDirCreatesWhatIsMissing(t *testing.T) {
	root := t.TempDir()
	fs := fsys.NewLocalStorage(root)

	target := "app/http/handlers"
	if err := ensurePackageDir(fs, target); err != nil {
		t.Fatalf("ensurePackageDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, target)); err != nil {
		t.Errorf("the directory was not created: %v", err)
	}
}

func TestEnsurePackageDirAcceptsAnExistingDirectory(t *testing.T) {
	root := t.TempDir()
	fs := fsys.NewLocalStorage(root)

	if err := ensurePackageDir(fs, "internal/models"); err != nil {
		t.Fatal(err)
	}
	if err := ensurePackageDir(fs, "internal/models"); err != nil {
		t.Errorf("a second call errored: %v", err)
	}
}

func TestEnsurePackageDirRejectsAnEmptyPath(t *testing.T) {
	if err := ensurePackageDir(fsys.NewLocalStorage(t.TempDir()), ""); err == nil {
		t.Error("ensurePackageDir() accepted an empty path")
	}
}

// Each generator must ask where its code goes rather than deciding for itself.
// They each hardcoded a directory, which meant the paths a project configures
// — and the utils helpers that read them — did nothing at all.
func TestGeneratorsUseTheProjectsPaths(t *testing.T) {
	seedPaths(t, projectPaths{
		HandlerPath:   "./configured/handlers",
		InputPath:     "./configured/inputs",
		ModelPath:     "./configured/models",
		MigrationPath: "./configured/migrations",
	})

	tests := map[string]struct {
		got  string
		want string
	}{
		"model":     {(&ModelGenerator{}).GetPackagePath(), "./configured/models"},
		"handler":   {(&HandlerGenerator{}).GetPackagePath(), "./configured/handlers"},
		"input":     {(&InputGenerator{}).GetPackagePath(), "./configured/inputs"},
		"migration": {(&MigrationGenerator{}).GetPackagePath(), "./configured/migrations"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("GetPackagePath() = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

// MIGRATIONS_DIR still wins, because the migrate command reads it too and the
// two have to agree about where migrations live.
func TestMigrationsDirOverridesTheConfiguredPath(t *testing.T) {
	seedPaths(t, projectPaths{MigrationPath: "./configured/migrations"})

	t.Setenv("MIGRATIONS_DIR", "./db/migrations")

	if got := (&MigrationGenerator{}).GetPackagePath(); got != "./db/migrations" {
		t.Errorf("GetPackagePath() = %q, want the environment override", got)
	}
}

// seedPaths installs a known set of paths for a test, standing in for the
// answer a project would have given. It consumes the sync.Once so that Paths
// returns what was seeded rather than resolving for real.
func seedPaths(t *testing.T, paths projectPaths) {
	t.Helper()
	pathsOnce = sync.Once{}
	pathsCached = paths
	pathsOnce.Do(func() {})
	t.Cleanup(func() { pathsOnce = sync.Once{} })
}
