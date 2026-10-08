package cmd

import (
	"fmt"
	"os"
	"sige/internal/api"
	"sige/internal/config"

	"github.com/spf13/cobra"
)

var apiPort string

func init() {
	rootCmd.AddCommand(apiCmd)
	apiCmd.Flags().StringVarP(&apiPort, "port", "p", "8080", "API server port")
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Starts the HTTP API server",
	Run: func(cmd *cobra.Command, args []string) {
		// Valida configuração na inicialização
		if _, err := config.Resolve("config.json"); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Configuration error: %v\n", err)
			os.Exit(1)
		}

		bootstrapPrivileged()

		fmt.Fprintf(os.Stderr, "[sige] Starting server on port %s...\n", apiPort)
		api.StartServer(apiPort)
	},
}
