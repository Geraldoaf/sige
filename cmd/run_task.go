package cmd

import (
	"fmt"
	"os"
	"sige/internal/config"
	"sige/internal/sandbox"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	memLimit        int64
	cpuLimit        string
	taskName        string
	timeoutSec      int
	tmpLimit        int
	fileLimitFlag   int
	nofileLimitFlag int
)

var runTaskCmd = &cobra.Command{
	Use:   "run-task [command]",
	Short: "Runs an isolated command inside a cgroup v2 sandbox",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		defaultCfg, err := config.LoadConfig("config.json")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: configuration file 'config.json' not found in current folder.\n")
			fmt.Fprintf(os.Stderr, "Please generate the default configurations first by running:\n\n")
			fmt.Fprintf(os.Stderr, "    sige init-config\n\n")
			os.Exit(1)
		}

		custom := config.CustomLimits{}
		if cmd.Flags().Changed("mem") {
			custom.MemoryMB = memLimit
		}
		if cmd.Flags().Changed("cpu") {
			custom.CPU = cpuLimit
		}
		if cmd.Flags().Changed("timeout") {
			custom.TimeoutSec = timeoutSec
		}
		if cmd.Flags().Changed("tmp-limit") {
			custom.TmpLimitMB = tmpLimit
		}
		if cmd.Flags().Changed("file-limit") {
			custom.MaxFileSizeMB = fileLimitFlag
		}
		if cmd.Flags().Changed("nofile-limit") {
			custom.MaxOpenFiles = nofileLimitFlag
		}

		resolved := config.MergeLimits(defaultCfg, custom)

		resolvedName := taskName
		if !cmd.Flags().Changed("name") {

			resolvedName = fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
		}

		sandboxConfig := sandbox.Config{
			Name:          resolvedName,
			MemoryMB:      resolved.MemoryMB,
			CPU:           resolved.CPU,
			TimeoutSec:    resolved.TimeoutSec,
			TmpLimitMB:    resolved.TmpLimitMB,
			MaxFileSizeMB: resolved.MaxFileSizeMB,
			MaxOpenFiles:  resolved.MaxOpenFiles,
		}

		memStr := fmt.Sprintf("%dMB", sandboxConfig.MemoryMB)
		if sandboxConfig.MemoryMB <= 0 {
			memStr = "unlimited"
		}

		cpuStr := sandboxConfig.CPU
		if cpuStr == "" {
			cpuStr = "unlimited"
		}

		tmpStr := fmt.Sprintf("%dMB", sandboxConfig.TmpLimitMB)
		if sandboxConfig.TmpLimitMB <= 0 {
			tmpStr = "read-only"
		}

		fileStr := fmt.Sprintf("%dMB", sandboxConfig.MaxFileSizeMB)
		if sandboxConfig.MaxFileSizeMB <= 0 {
			fileStr = "unlimited"
		}

		fmt.Printf("Starting sandbox '%s' (%s RAM, CPU: %s, Timeout: %ds, /tmp: %s, FileLimit: %s, OpenFiles: %d)...\n",
			sandboxConfig.Name, memStr, cpuStr, sandboxConfig.TimeoutSec, tmpStr, fileStr, sandboxConfig.MaxOpenFiles)

		taskCmd := args[0]
		taskArgs := args[1:]

		fmt.Printf("Executing: %s %s\n", taskCmd, strings.Join(taskArgs, " "))

		result, err := sandbox.Execute(sandboxConfig, taskCmd, taskArgs)

		memPeakStr := config.FormatBytes(result.MemoryPeak)

		fmt.Printf("\n--- Sandbox Execution Summary ---\n")
		fmt.Printf("Termination Status: %s\n", strings.ToUpper(result.Status))
		fmt.Printf("Exit Code         : %d\n", result.ExitCode)

		durationLimit := "unlimited"
		if result.LimitTimeoutSec > 0 {
			durationLimit = fmt.Sprintf("%ds", result.LimitTimeoutSec)
		}
		fmt.Printf("Elapsed Time      : %v (Limit: %s)\n", result.Duration.Round(time.Millisecond), durationLimit)
		fmt.Printf("CPU Time          : User: %v, Sys: %v\n", result.CPUUserTime.Round(time.Microsecond), result.CPUSystemTime.Round(time.Microsecond))

		memoryLimit := "unlimited"
		if result.LimitMemoryBytes > 0 {
			memoryLimit = config.FormatBytes(result.LimitMemoryBytes)
		}
		fmt.Printf("Peak RAM Usage    : %s (Limit: %s)\n", memPeakStr, memoryLimit)

		fileSizeLimit := "unlimited"
		if result.LimitFileSizeMax > 0 {
			fileSizeLimit = config.FormatBytes(result.LimitFileSizeMax)
		}
		fmt.Printf("File Size Limit   : %s\n", fileSizeLimit)
		fmt.Printf("Open Files Limit  : %d descriptors\n", result.LimitOpenFiles)
		fmt.Printf("----------------------------------\n")

		if result.Stderr != "" {
			fmt.Printf("\n--- Standard Error (Stderr) ---\n%s---------------------------------\n", result.Stderr)
		}
		if result.Stdout != "" {
			fmt.Printf("\n--- Standard Output (Stdout) ---\n%s---------------------------------\n", result.Stdout)
		}

		if err != nil {
			fmt.Printf("\nProcess Error: %v\n", err)
			if result.Status == "oom" {
				fmt.Println("Hint: The process was killed due to memory exhaustion (Out-Of-Memory Killer).")
			} else if result.Status == "timeout" {
				fmt.Println("Hint: The process was killed because it exceeded the timeout limit.")
			} else if result.Status == "file_size_exceeded" {
				fmt.Println("Hint: The process was killed because it attempted to write a file exceeding the maximum size limit.")
			}
		}
	},
}

func init() {
	runTaskCmd.Flags().Int64VarP(&memLimit, "mem", "m", 50, "Memory limit in MB")
	runTaskCmd.Flags().StringVarP(&cpuLimit, "cpu", "c", "", "CPU limit in % (e.g. '10') or in cgroup format (e.g. '10000 100000')")
	runTaskCmd.Flags().StringVarP(&taskName, "name", "n", "sige-sandbox", "Sandbox name (cgroup slice)")
	runTaskCmd.Flags().IntVarP(&timeoutSec, "timeout", "t", 0, "Time limit in seconds (0 = unlimited)")
	runTaskCmd.Flags().IntVarP(&tmpLimit, "tmp-limit", "", 64, "Size limit for /tmp in MB (0 = read-only)")
	runTaskCmd.Flags().IntVarP(&fileLimitFlag, "file-limit", "", 15, "Maximum file size limit in MB (0 = unlimited)")
	runTaskCmd.Flags().IntVarP(&nofileLimitFlag, "nofile-limit", "", 256, "Maximum open files/sockets limit (0 = unlimited)")

	rootCmd.AddCommand(runTaskCmd)
}
