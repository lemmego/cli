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
	Version: "0.1.53",
}

// AddCmd adds a new sub-command to the root command.
func AddCmd(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
}

// Execute the command and register the sub-commands.
func Execute() error {
	newCmd.Flags().BoolVar(&enableExperimental, "exp", false, "Enable experimental features (GPA)")
	newCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Create a project without prompts")
	newCmd.Flags().StringVar(&projectModule, "module", "", "Go module path for the new project")
	newCmd.Flags().StringVar(&projectPreset, "preset", "", "Project preset: mvc or rest_api")
	newCmd.Flags().StringVar(&projectORM, "orm", "", "SQL ORM: orm (Lemmego), gorm or bun")
	newCmd.Flags().StringVar(&projectFrontend, "frontend", "", "MVC frontend preset")
	newCmd.Flags().BoolVar(&projectRedis, "redis", false, "Enable Redis")
	newCmd.Flags().BoolVar(&projectAuth, "auth", false, "Enable authentication")
	newCmd.Flags().BoolVar(&projectGPA, "gpa", false, "Enable GPA")
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
