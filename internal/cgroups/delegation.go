package cgroups

import (
	"os"
	"strings"
)

// SetupDelegation move processos para /init e habilita controladores cpu, memory e pids na raiz do cgroups.
func SetupDelegation() {
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.subtree_control"); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile("/sys/fs/cgroup/cgroup.subtree_control")
	if err != nil {
		return
	}

	enabled := make(map[string]bool)
	for _, controller := range strings.Fields(string(data)) {
		enabled[controller] = true
	}
	if enabled["cpu"] && enabled["memory"] && enabled["pids"] {
		return
	}

	if err := os.MkdirAll("/sys/fs/cgroup/init", 0755); err != nil {
		return
	}

	pidsData, err := os.ReadFile("/sys/fs/cgroup/cgroup.procs")
	if err != nil {
		return
	}

	pids := strings.Split(string(pidsData), "\n")
	for _, pidStr := range pids {
		pidStr = strings.TrimSpace(pidStr)
		if pidStr == "" {
			continue
		}

		_ = os.WriteFile("/sys/fs/cgroup/init/cgroup.procs", []byte(pidStr), 0644)
	}

	_ = os.WriteFile("/sys/fs/cgroup/cgroup.subtree_control", []byte("+cpu +memory +pids"), 0644)
}
