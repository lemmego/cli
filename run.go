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
	Use:                "run [args]",
	Short:              "Run the Lemmego application with optional arguments",
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		if !isLemmegoProject() {
			fmt.Println("Error: This does not appear to be a Lemmego project directory.")
			return
		}

		// Only build frontend assets for commands that serve HTTP
		if needsFrontend(args) {
			if hasTemplFiles() {
				EnsureBinary("templ")
				fmt.Println("> Generating templ files...")
				RunCommand(".", "templ", "generate")
			}
			if fileExists("package.json") {
				EnsureBinary("node")
				fmt.Println("> Building frontend assets...")
				RunCommand(".", npmBinary(), "run", "build")
			}
		}

		// Run the application — args includes the subcommand and its flags
		goRunCmd := exec.Command("go", append([]string{"run", "./cmd/app"}, args...)...)
		goRunCmd.Stdout = os.Stdout
		goRunCmd.Stderr = os.Stderr

		if err := goRunCmd.Run(); err != nil {
			fmt.Printf("Error running the app: %v\n", err)
		}
	},
}

func needsFrontend(args []string) bool {
	if len(args) == 0 {
		return true
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if strings.Contains(a, ":") {
			return false
		}
		switch a {
		case "inspire", "key", "migrate", "rollback":
			return false
		}
		break
	}
	return true
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

// GetModuleName reads the go.mod file and returns the module name
func GetModuleName() (string, error) {
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
