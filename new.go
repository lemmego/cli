package cli

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var enableExperimental bool
var nonInteractive bool

// projectFlagSet groups the flags `new` accepts.
//
// They are a struct rather than nine package-level variables so that adding
// one does not mean editing every test's save-and-restore block, which is
// what kept them from being added.
type projectFlagSet struct {
	Module   string
	Preset   string
	Frontend string

	ORM      string
	Database string
	Cache    string
	Queue    string
	Session  string
	Disk     string

	Redis bool
	Auth  bool
	GPA   bool
	MCP   bool
}

var projectFlags projectFlagSet

var newCmd = &cobra.Command{
	Use:     "new [dirname]",
	Aliases: []string{"n"},
	Short:   "Create an app",
	Long:    `Create a new Lemmego app`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dirname := args[0]
		dirPath := DirPath(dirname)

		var cfg *ProjectConfig
		if nonInteractive {
			var err error
			cfg, err = collectNonInteractiveProjectConfig(dirname)
			if err != nil {
				return err
			}
		} else {
			cfg = collectProjectConfig(dirname, enableExperimental)
			if cfg == nil {
				return nil
			}
		}

		if err := ensureEmptyDir(dirname); err != nil {
			return err
		}

		if err := ScaffoldProject(*cfg, dirPath); err != nil {
			log.Fatal("Error scaffolding project:", err)
		}

		if err := renameModule(cfg.ModuleName, dirPath); err != nil {
			return err
		}
		copyEnvFile(dirPath)
		createSQLiteDatabase(dirPath)

		if hasNodeDeps(*cfg) {
			EnsureBinary("node")
			installNodeModules(dirPath)
			buildFrontend(dirPath)
		}

		// templ generate has to run before the modules are tidied: it writes
		// the _templ.go files that make templates a Go package, and without
		// them tidy cannot resolve the project's own import of it.
		if hasTemplGenerate(*cfg) {
			fmt.Println("> Generating templ files...")
			EnsureBinary("templ")
			RunCommand(dirPath, "templ", "generate")
		}

		installGoModules(dirPath)
		generateAppKey(dirPath)

		fmt.Printf("\nSuccessfully created a new Lemmego app with module name: %s in directory: %s\n", cfg.ModuleName, dirname)
		fmt.Println("> Navigate to your new project, and run:")
		fmt.Println("cd", dirname)
		fmt.Println("lemmego dev")
		return nil
	},
}

func installGoModules(dirPath string) {
	fmt.Println("> Installing go modules...")
	RunCommand(dirPath, "go", "mod", "tidy")
	fmt.Println("Done")
}

func installNodeModules(dirPath string) {
	fmt.Println("> Installing node modules...")
	RunCommand(dirPath, npmBinary(), "install")
}

func buildFrontend(dirPath string) {
	fmt.Println("> Building frontend assets...")
	RunCommand(dirPath, npmBinary(), "run", "build")
}

func createSQLiteDatabase(dirPath string) {
	storageDir := filepath.Join(dirPath, "storage")
	databaseFile := filepath.Join(storageDir, "database.sqlite")

	os.MkdirAll(storageDir, 0755)

	dbFile, err := os.Create(databaseFile)
	if err != nil {
		fmt.Println("Warning: Could not create SQLite database file:", err)
	} else {
		dbFile.Close()
		fmt.Println("> Creating SQLite db in ./storage/database.sqlite")
	}
}

func copyEnvFile(dirPath string) {
	sourceFile := filepath.Join(dirPath, ".env.example")
	destFile := filepath.Join(dirPath, ".env")
	if _, err := os.Stat(sourceFile); err == nil {
		fmt.Println("> Copying .env.example to .env...")
		if err := CopyFile(sourceFile, destFile); err != nil {
			fmt.Println("Warning: Could not copy .env.example to .env:", err)
		}
	} else if !os.IsNotExist(err) {
		fmt.Println("Warning: Error checking for .env.example file:", err)
	}
}

// generateAppKey writes a signing key into the new project's .env.
//
// Done here rather than by shelling out to `lemmego run appkey`, which cannot
// work: `run` boots the generated application to reach its command, and the
// application cannot boot until somebody has filled in database credentials.
// That ordering leaves a new project with no key, and an application with no
// key cannot authenticate anybody — which is the first thing somebody tries.
//
// The project keeps its own `appkey` command for rotating the key later, when
// the database does work. This is only the first one.
func generateAppKey(dirPath string) {
	fmt.Println("> Generating app key...")

	// Thirty-two random bytes, base64 encoded — the same thing
	// api/utils.GenerateKey produces. Reimplemented over the standard library
	// rather than imported, because the CLI does not depend on the framework
	// and should not start: api/app reaches 300 packages, and this is four
	// lines of crypto/rand.
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		fmt.Println("Warning: could not generate an app key:", err)
		return
	}
	envPath := filepath.Join(dirPath, ".env")
	if err := setEnvValue(envPath, "APP_KEY", base64.StdEncoding.EncodeToString(key)); err != nil {
		fmt.Println("Warning: could not write the app key:", err)
		fmt.Println("Run `make appkey` once the database is configured.")
	}
}

// setEnvValue replaces the assignment of key in a .env file, or appends one.
//
// Matching is anchored to "<key>=" at the start of a trimmed line, never a
// substring: the old project-side version used strings.Contains and so
// overwrote the comment explaining APP_KEY instead of the variable, wedging
// the key between two halves of a sentence.
func setEnvValue(path, key, value string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	prefix := key + "="
	lines := strings.Split(string(contents), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[i] = prefix + value
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600)
		}
	}

	// No assignment to replace, so add one rather than reporting success
	// having done nothing.
	appended := string(contents)
	if !strings.HasSuffix(appended, "\n") {
		appended += "\n"
	}
	appended += prefix + value + "\n"
	return os.WriteFile(path, []byte(appended), 0o600)
}

func renameModule(newModuleName string, dirPath string) error {
	fmt.Println("> Applying module name...")
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".templ") || filepath.Base(path) == "go.mod") {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			newContent := strings.ReplaceAll(string(content), "github.com/lemmego/lemmego", newModuleName)
			if err = os.WriteFile(path, []byte(newContent), 0644); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("replacing module name: %w", err)
	}

	// Rewriting the module path reorders imports, so the tree is formatted
	// after the rename rather than before it.
	if err := formatGoFiles(dirPath); err != nil {
		return fmt.Errorf("formatting generated code: %w", err)
	}
	return nil
}

func npmBinary() string {
	if HasBinary("pnpm") {
		return "pnpm"
	} else if HasBinary("yarn") {
		return "yarn"
	} else if HasBinary("npm") {
		return "npm"
	} else {
		log.Fatal("You need to install pnpm, yarn or npm")
	}
	return ""
}
