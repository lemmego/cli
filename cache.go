package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cacheCleanCmd = &cobra.Command{
	Use:   "cache-clean",
	Short: "Clear the local scaffold cache",
	Long:  `Removes the cached scaffold templates at ~/.cache/lemmego/scaffold, forcing the CLI to use the embedded scaffold on the next project creation.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir := scaffoldCache()
		if cacheDir == "" {
			fmt.Println("Could not determine home directory.")
			return
		}

		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			fmt.Println("Cache directory does not exist. Nothing to clean.")
			return
		}

		if err := os.RemoveAll(cacheDir); err != nil {
			fmt.Printf("Error clearing cache: %v\n", err)
			return
		}

		fmt.Printf("Cleared scaffold cache at %s\n", cacheDir)
	},
}
