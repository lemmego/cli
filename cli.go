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
	Use:     "new [module-name]",
	Aliases: []string{"n"},
	Short:   "Create an app",
	Long:    `Create a new Lemmego app by cloning the repository and replacing the module name`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		newModuleName := args[0]

		// Check if git is installed
		if _, err := exec.LookPath("git"); err != nil {
			fmt.Println("Error: Git must be installed to use this command.")
			return
		}

		// Check if the current directory is empty
		dir, _ := os.Getwd()
		files, _ := filepath.Glob(filepath.Join(dir, "*"))
		if len(files) > 0 {
			fmt.Println("Error: The current directory must be empty to create a new project.")
			return
		}

		fmt.Println("> Cloning repository...")
		_, err := git.PlainClone(dir, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: os.Stdout,
		})
		if err != nil {
			fmt.Println("Error cloning repository:", err)
			return
		}

		fmt.Println("> Replacing module name in .go, .templ, and go.mod files...")
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
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
				fmt.Printf("  - Updated: %s\n", path)
			}
			return nil
		})
		if err != nil {
			fmt.Println("Error replacing module name:", err)
			return
		}

		// Remove .git directory
		fmt.Println("> Cleaning up repository metadata...")
		if err := os.RemoveAll(filepath.Join(dir, ".git")); err != nil {
			fmt.Println("Warning: Unable to remove .git directory:", err)
		}

		fmt.Println("\nSuccessfully created a new Lemmego app with module name:", newModuleName)
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
}
