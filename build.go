package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build frontend assets",
	Run: func(cmd *cobra.Command, args []string) {
		if !isLemmegoProject() {
			fmt.Println("Error: This does not appear to be a Lemmego project directory.")
			return
		}

		if hasTemplFiles() {
			fmt.Println("> Generating templ files...")
			EnsureBinary("templ")
			RunCommand(".", "templ", "generate")
		}

		if fileExists("package.json") {
			EnsureBinary("node")
			fmt.Println("> Building frontend assets...")
			RunCommand(".", npmBinary(), "run", "build")
		}

		if !hasTemplFiles() && !fileExists("package.json") {
			fmt.Println("Nothing to build (no templ files or Node dependencies found).")
		}
	},
}
