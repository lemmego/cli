package cli

import (
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

var shouldRunInteractively = false
var registerOnce sync.Once

// rootCmd is the top-level command, which will
// hold all the subcommands such as gen, or any package-level
// commands installed via service providers.
var rootCmd = &cobra.Command{
	Use:     "",
	Short:   fmt.Sprintf("%s", os.Getenv("APP_NAME")),
	Version: resolveVersion(),
}

// AddCmd adds a new sub-command to the root command.
func AddCmd(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

// Execute the command and register the sub-commands.
func Execute() error {
	registerOnce.Do(registerCommands)
	return rootCmd.Execute()
}

func registerCommands() {
	newCmd.Flags().BoolVar(&enableExperimental, "exp", false, "Enable experimental features (GPA)")
	newCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Create a project without prompts")
	newCmd.Flags().StringVar(&projectFlags.Module, "module", "", "Go module path for the new project")
	newCmd.Flags().StringVar(&projectFlags.Preset, "preset", "", "Project preset: mvc or rest_api")
	newCmd.Flags().StringVar(&projectFlags.Frontend, "frontend", "", "MVC frontend preset")
	newCmd.Flags().StringVar(&projectFlags.Database, "database", "", "Database: sqlite, mysql, postgres or none")
	newCmd.Flags().StringVar(&projectFlags.ORM, "orm", "", "SQL layer: orm (Lemmego), gorm, bun or none")
	newCmd.Flags().StringVar(&projectFlags.Cache, "cache", "", "Cache: file, memory, redis or none")
	newCmd.Flags().StringVar(&projectFlags.Queue, "queue", "", "Queue: sql, redis or none")
	newCmd.Flags().StringVar(&projectFlags.Session, "session", "", "Sessions: file, memory, redis or database")
	newCmd.Flags().StringVar(&projectFlags.Disk, "disk", "", "Default storage disk: local or s3")
	newCmd.Flags().BoolVar(&projectFlags.Redis, "redis", false, "Use Redis for anything left unset")
	newCmd.Flags().BoolVar(&projectFlags.Auth, "auth", true, "Scaffold authentication")
	newCmd.Flags().BoolVar(&projectFlags.GPA, "gpa", false, "Enable GPA")
	newCmd.Flags().BoolVar(&projectFlags.MCP, "mcp", true,
		"Register the MCP server, so an AI coding agent can read this application's state")
	genCmd.PersistentFlags().BoolVarP(&shouldRunInteractively, "interactive", "i", false, "Run interactively")

	mcpInstallCmd.Flags().StringSliceVar(&mcpAgents, "agent", nil,
		"Configure only these agents (claude-code, cursor, codex, copilot, windsurf); repeatable")
	mcpInstallCmd.Flags().BoolVar(&mcpNoSkills, "no-skills", false, "Do not write .ai/skills")
	mcpInstallCmd.Flags().BoolVar(&mcpNoRules, "no-rules", false, "Do not create .ai/rules")
	mcpInstallCmd.Flags().BoolVar(&mcpNoGuides, "no-guidelines", false, "Do not write the guideline block")

	genCmd.AddCommand(handlerCmd)
	genCmd.AddCommand(migrationCmd)
	genCmd.AddCommand(modelCmd)
	genCmd.AddCommand(inputCmd)
	genCmd.AddCommand(formCmd)

	AddCmd(newCmd)
	AddCmd(runCmd)
	AddCmd(devCmd)
	AddCmd(buildCmd)
	AddCmd(genCmd)
	AddCmd(inertiaSSRCmd)
	AddCmd(cacheCleanCmd)
	AddCmd(mcpInstallCmd)
}
