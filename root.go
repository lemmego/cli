package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var shouldRunInteractively = false

// rootCmd is the top-level command, which will
// hold all the subcommands such as gen, or any package-level
// commands installed via service providers.
var rootCmd = &cobra.Command{
	Use:     "",
	Short:   fmt.Sprintf("%s", os.Getenv("APP_NAME")),
	Version: "0.1.57",
}

// AddCmd adds a new sub-command to the root command.
func AddCmd(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

// Execute the command and register the sub-commands.
func Execute() error {
	newCmd.Flags().BoolVar(&enableExperimental, "exp", false, "Enable experimental features (GPA)")
	newCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Create a project without prompts")
	newCmd.Flags().StringVar(&projectFlags.Module, "module", "", "Go module path for the new project")
	newCmd.Flags().StringVar(&projectFlags.Preset, "preset", "", "Project preset: mvc or rest_api")
	newCmd.Flags().StringVar(&projectFlags.Frontend, "frontend", "", "MVC frontend preset")
	newCmd.Flags().StringVar(&projectFlags.Database, "database", "", "Database: sqlite, mysql, postgres or none")
	newCmd.Flags().StringVar(&projectFlags.ORM, "orm", "", "SQL layer: orm (Lemmego), gorm, bun or none")
	newCmd.Flags().StringVar(&projectFlags.Cache, "cache", "", "Cache: file, memory, redis or none")
	newCmd.Flags().StringVar(&projectFlags.Queue, "queue", "", "Queue: sql, redis or none")
	newCmd.Flags().StringVar(&projectFlags.Session, "session", "", "Sessions: file, memory or redis")
	newCmd.Flags().StringVar(&projectFlags.Disk, "disk", "", "Default storage disk: local or s3")
	newCmd.Flags().BoolVar(&projectFlags.Redis, "redis", false, "Use Redis for anything left unset")
	newCmd.Flags().BoolVar(&projectFlags.Auth, "auth", true, "Scaffold authentication")
	newCmd.Flags().BoolVar(&projectFlags.GPA, "gpa", false, "Enable GPA")
	genCmd.PersistentFlags().BoolVarP(&shouldRunInteractively, "interactive", "i", false, "Run interactively")

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

	return rootCmd.Execute()
}
