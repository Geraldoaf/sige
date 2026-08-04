package sandbox

import (
	"fmt"
	"regexp"

	"sige/internal/cgroups"
)

func Execute(config Config, command string, args []string) (ExecutionResult, error) {

	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, config.Name)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error validating sandbox name: %w", err)
	}
	if !matched {
		return ExecutionResult{}, fmt.Errorf("invalid sandbox name '%s': must contain only alphanumeric characters, hyphens, and underscores", config.Name)
	}

	if !cgroups.VerifyCgroupsVersion() {
		return ExecutionResult{}, fmt.Errorf("system does not support cgroup v2 or is not in Unified Mode")
	}

	memBytes := config.GetMemoryBytes()
	cpuFormatted := config.GetFormattedCPU()

	mgr, err := cgroups.CreateCgroup(config.Name, memBytes, cpuFormatted)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error creating cgroup: %w", err)
	}

	defer cgroups.DeleteCgroup(mgr)

	return Run(mgr, config, command, args...)
}
