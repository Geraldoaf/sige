package cmd

import (
	"fmt"
	"sige/internal/cgroups"

	"github.com/spf13/cobra"
)

var verifyCgroupsCmd = &cobra.Command{
	Use:   "cgroups-version",
	Short: "Verifica a versão do cgroups no sistema",
	Run: func(cmd *cobra.Command, args []string) {
		mode := cgroups.GetCgroupsMode()
		fmt.Printf("cgroup version: %s\n", mode)
	},
}

func init() {
	rootCmd.AddCommand(verifyCgroupsCmd)
}
