package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const lemmegoIndicator = "github.com/lemmego/api/app"

var runCmd = &cobra.Command{
	Use:   "run [args]",
	Short: "Run the Lemmego application with optional arguments",
	Run: func(cmd *cobra.Command, args []string) {
		// Check if we're in a Lemmego project
		if !isLemmegoProject() {
			fmt.Println("Error: This does not appear to be a Lemmego project directory.")
			return
		}

		// Construct the go run command
		goRunCmd := exec.Command("go", append([]string{"run", "./cmd/app"}, args...)...)

		// Set up stdout and stderr to be the same as the parent's
		goRunCmd.Stdout = os.Stdout
		goRunCmd.Stderr = os.Stderr

		// Execute the command
		if err := goRunCmd.Run(); err != nil {
			fmt.Printf("Error running the app: %v\n", err)
			return
		}
	},
}

// Function to check if the current directory is a Lemmego project
func isLemmegoProject() bool {
	mainGoPath := filepath.Join("./cmd/app", "main.go")
	content, err := os.ReadFile(mainGoPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), lemmegoIndicator)
}

// getModuleName reads the go.mod file and returns the module name
func getModuleName() (string, error) {
	file, err := os.Open("go.mod")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			// Extract everything after "module "
			moduleName := strings.TrimSpace(line[7:]) // 7 is len("module ")
			return moduleName, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}
