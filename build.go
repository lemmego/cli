package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// viteHotFile is written by the Vite dev server and must never survive into a
// build.
const viteHotFile = "public/hot"

// hasInertiaSSREntry reports whether the project has an SSR entry point.
func hasInertiaSSREntry() bool {
	for _, name := range []string{"ssr.tsx", "ssr.jsx", "ssr.ts", "ssr.js"} {
		if fileExists(filepath.Join("resources", "js", name)) {
			return true
		}
	}
	return false
}

// hasNpmScript reports whether package.json defines a script.
func hasNpmScript(name string) bool {
	body, err := os.ReadFile("package.json")
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(body, &pkg); err != nil {
		return false
	}
	return pkg.Scripts[name] != ""
}

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

			// The SSR bundle is a separate build, and not running it is why
			// "the bundle has never been built" was the normal state of every
			// Inertia project: the documented build command never built it.
			//
			// Only when the project has an SSR entry and a script for it, so
			// a client-only project is untouched.
			if hasInertiaSSREntry() && hasNpmScript("build:ssr") {
				fmt.Println("> Building the SSR bundle...")
				RunCommand(".", npmBinary(), "run", "build:ssr")
			}
		}

		// Last, and unconditionally. public/hot points both the asset tags
		// and the server-side renderer at a Vite dev server; left behind in a
		// build it produces a deploy whose CSS and JS 404 and whose markup is
		// never server-rendered, with nothing in the logs to say why.
		if fileExists(viteHotFile) {
			if err := os.Remove(viteHotFile); err != nil {
				fmt.Printf("Warning: could not remove %s: %v\n", viteHotFile, err)
			} else {
				fmt.Printf("> Removed %s\n", viteHotFile)
			}
		}

		if !hasTemplFiles() && !fileExists("package.json") {
			fmt.Println("Nothing to build (no templ files or Node dependencies found).")
		}
	},
}
