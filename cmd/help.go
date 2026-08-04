package cmd

import (
	"fmt"
	"os"
	"sige/internal/cgroups"
	"sige/internal/cli"

	"github.com/spf13/cobra"
)

var cliInput string

var rootCmd = &cobra.Command{
	Use:   "sige",
	Short: "SIGE - Sistema de Isolamento e Gerenciamento de Execução",
	Long:  `SIGE: Sandbox segura para execução de código não confiável via cgroups v2, namespaces e seccomp.`,
	Run: func(cmd *cobra.Command, args []string) {
		if cliInput != "" {
			cli.Run(cliInput)
		} else {
			_ = cmd.Help()
		}
	},
}

func init() {
	rootCmd.Flags().StringVarP(&cliInput, "input", "i", "", "CLI terminal mode command input")
}

func Execute() {
	cgroups.SetupDelegation()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
