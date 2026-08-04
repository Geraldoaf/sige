package cmd

import (
	"fmt"
	"os"
	"sige/internal/config"

	"github.com/spf13/cobra"
)

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Generates the default configuration file config.json",
	Run: func(cmd *cobra.Command, args []string) {
		const filename = "config.json"

		if _, err := os.Stat(filename); err == nil {
			fmt.Fprintf(os.Stderr, "Error: The file '%s' already exists in the current folder. To avoid overwriting your configurations, the operation was cancelled.\n", filename)
			os.Exit(1)
		}

		err := config.GenerateDefaultConfig(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating configuration file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Default configuration file '%s' successfully generated in current folder!\n", filename)
	},
}

func init() {
	rootCmd.AddCommand(initConfigCmd)
}
