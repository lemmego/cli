package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/spf13/cobra"
)

const repoURL = "https://github.com/lemmego/lemmego"

var newCmd = &cobra.Command{
	Use:     "new [dirname]",
	Aliases: []string{"n"},
	Short:   "Create an app",
	Long:    `Create a new Lemmego app`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dirname := args[0]

		// Check if git is installed
		if _, err := exec.LookPath("git"); err != nil {
			fmt.Println("Error: Git must be installed to use this command.")
			return
		}

		// Get current working directory
		currentDir, _ := os.Getwd()
		dirPath := filepath.Join(currentDir, dirname)

		// Check if directory exists
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			// If directory doesn't exist, create it
			fmt.Printf("> Creating new directory: %s\n", dirname)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				fmt.Println("Error creating directory:", err)
				return
			}
		} else {
			// Directory exists, check if it's empty
			files, _ := filepath.Glob(filepath.Join(dirPath, "*"))
			if len(files) > 0 {
				fmt.Println("Error: The directory must be empty to create a new project.")
				return
			}
		}

		// Prompt for module name
		fmt.Print("Enter the module name: ")
		var newModuleName string
		_, err := fmt.Scanln(&newModuleName)
		if err != nil {
			fmt.Println("Error reading module name:", err)
			return
		}

		fmt.Println("> Downloading starter template...")
		_, err = git.PlainClone(dirPath, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: os.Stdout,
		})
		if err != nil {
			fmt.Println("Error cloning repository:", err)
			return
		}

		fmt.Println("> Creating a new project, please wait...")
		err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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
			fmt.Println("Error replacing module name:", err)
			return
		}

		// Copy .env.example to .env
		sourceFile := filepath.Join(dirPath, ".env.example")
		destFile := filepath.Join(dirPath, ".env")
		if _, err := os.Stat(sourceFile); err == nil {
			fmt.Println("> Copying .env.example to .env...")
			if err := copyFile(sourceFile, destFile); err != nil {
				fmt.Println("Warning: Could not copy .env.example to .env:", err)
			}
		} else if !os.IsNotExist(err) {
			fmt.Println("Warning: Error checking for .env.example file:", err)
		}

		// Remove .git directory
		fmt.Println("> Cleaning up repository metadata...")
		if err := os.RemoveAll(filepath.Join(dirPath, ".git")); err != nil {
			fmt.Println("Warning: Unable to remove .git directory:", err)
		}

		// Create SQLite database file
		storageDir := filepath.Join(dirPath, "storage")
		databaseFile := filepath.Join(storageDir, "database.sqlite")

		// Ensure storage directory exists
		if err := os.MkdirAll(storageDir, 0755); err != nil {
			fmt.Println("Warning: Could not create storage directory:", err)
		} else {
			// Create the database file
			dbFile, err := os.Create(databaseFile)
			if err != nil {
				fmt.Println("Warning: Could not create SQLite database file:", err)
			} else {
				dbFile.Close() // Ensure the file is closed after creation
				fmt.Println("> Creating SQLite db in ./storage/database.sqlite")
			}
		}

		fmt.Println("\nSuccessfully created a new Lemmego app with module name:", newModuleName, "in directory:", dirname)
		fmt.Println("> Navigate to your new project, update the .env file, and run:")
		fmt.Println("cd", dirname)
		fmt.Println("go run ./cmd/app")
	},
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = destination.ReadFrom(source)
	return err
}
