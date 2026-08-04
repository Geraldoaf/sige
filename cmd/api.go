package cmd

import (
	"fmt"
	"os"
	"sige/internal/api"

	"github.com/spf13/cobra"
)

var apiPort string

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().StringVarP(&apiPort, "port", "p", "8080", "API server port")
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Starts the API server",
	Run: func(cmd *cobra.Command, args []string) {

		if _, err := os.Stat("config.json"); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: configuration file 'config.json' not found in current folder.\n")
			fmt.Fprintf(os.Stderr, "Please generate the default configurations first by running:\n\n")
			fmt.Fprintf(os.Stderr, "    sige init-config\n\n")
			os.Exit(1)
		}

		fmt.Printf("Starting API on port %s...\n", apiPort)
		api.StartServer(apiPort)
	},
}
