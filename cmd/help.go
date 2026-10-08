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
	Short: "SIGE - Execution Isolation and Management System",
	Long:  `SIGE: Secure sandbox for executing untrusted code via cgroups v2, namespaces, and seccomp.`,
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

// Execute inicializa a delegação do cgroups e executa a linha de comando raiz do Cobra.
func Execute() {
	cgroups.SetupDelegation()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}
