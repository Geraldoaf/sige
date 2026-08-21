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
	Short: "Inicia o servidor HTTP da API",
	Run: func(cmd *cobra.Command, args []string) {
		// Valida configuração na inicialização
		if _, err := config.Resolve("config.json"); err != nil {
			fmt.Fprintf(os.Stderr, "[sige] Erro de configuração: %v\n", err)
			os.Exit(1)
		}

		bootstrapPrivileged()

		fmt.Fprintf(os.Stderr, "[sige] Iniciando servidor na porta %s...\n", apiPort)
		api.StartServer(apiPort)
	},
}
