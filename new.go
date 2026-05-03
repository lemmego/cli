package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:     "new [dirname]",
	Aliases: []string{"n"},
	Short:   "Create an app",
	Long:    `Create a new Lemmego app`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dirname := args[0]
		dirPath := DirPath(dirname)

		cfg := collectProjectConfig(dirname)

		EnsureEmptyDir(dirname)

		if err := ScaffoldProject(cfg, dirPath); err != nil {
			log.Fatal("Error scaffolding project:", err)
		}

		renameModule(cfg.ModuleName, dirPath)
		copyEnvFile(dirPath)
		createSQLiteDatabase(dirPath)

		if hasNodeDeps(cfg) {
			EnsureBinary("node")
			installNodeModules(dirPath)
			buildFrontend(dirPath)
		}

		installGoModules(dirPath)

		if hasTemplGenerate(cfg) {
			fmt.Println("> Generating templ files...")
			RunCommand(dirPath, "templ", "generate")
		}

		fmt.Printf("\nSuccessfully created a new Lemmego app with module name: %s in directory: %s\n", cfg.ModuleName, dirname)
		fmt.Println("> Navigate to your new project, and run:")
		fmt.Println("cd", dirname)
		fmt.Println("lemmego run")
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

func renameModule(newModuleName string, dirPath string) {
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
		log.Fatal("Error replacing module name:", err)
	}
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
