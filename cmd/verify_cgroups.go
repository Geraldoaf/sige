package cmd

import (
	"fmt"
	"sige/internal/cgroups"

	"github.com/spf13/cobra"
)

var verifyCgroupsCmd = &cobra.Command{
	Use:   "cgroups-version",
	Short: "Verifies system cgroups version",
	Run: func(cmd *cobra.Command, args []string) {
		mode := cgroups.GetCgroupsMode()
		fmt.Printf("cgroup version: %s\n", mode)
	},
}

func init() {
	rootCmd.AddCommand(verifyCgroupsCmd)
}
