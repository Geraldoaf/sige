package cgroups

import (
	"fmt"
	"os"
	"strings"
	"time"

	"sige/internal/constants"

	"github.com/containerd/cgroups/v3/cgroup2"
)

// ensureSigeSubtreeControlDelegated habilita os controladores cpu, memory e pids no subtree_control.
func ensureSigeSubtreeControlDelegated() {
	path := "/sys/fs/cgroup" + constants.CgroupPrefix + "cgroup.subtree_control"

	data, err := os.ReadFile(path)
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

	_ = os.WriteFile(path, []byte("+cpu +memory +pids"), 0644)
}

// CreateCgroup cria um cgroup v2 com os limites de recursos especificados.
func CreateCgroup(name string, memLimitBytes int64, cpuMax string) (*cgroup2.Manager, error) {
	ensureSigeSubtreeControlDelegated()

	pidLimit := int64(constants.DefaultPIDLimit)
	res := cgroup2.Resources{
		Pids: &cgroup2.Pids{Max: pidLimit},
	}

	if memLimitBytes > 0 {
		swapLimit := int64(0)
		res.Memory = &cgroup2.Memory{
			Max:  &memLimitBytes,
			Swap: &swapLimit,
		}
	}

	if cpuMax != "" {
		res.CPU = &cgroup2.CPU{
			Max: cgroup2.CPUMax(cpuMax),
		}
	}

	mgr, err := cgroup2.NewManager("/sys/fs/cgroup", constants.CgroupPrefix+name, &res)
	if err != nil {
		return nil, fmt.Errorf("error creating cgroup: %w", err)
	}

	return mgr, nil
}

// JoinCgroup anexa o processo atual ao cgroup especificado.
func JoinCgroup(path string) error {
	return AddProcessToCgroup(path, 0)
}

// AddProcessToCgroup adiciona um PID específico ao cgroup.
func AddProcessToCgroup(path string, pid int) error {
	data := []byte(fmt.Sprintf("%d", pid))
	return os.WriteFile("/sys/fs/cgroup"+path+"/cgroup.procs", data, 0644)
}

// DeleteCgroup remove o cgroup gerenciado.
func DeleteCgroup(mgr *cgroup2.Manager) error {
	return mgr.Delete()
}

// KillAndDeleteCgroup encerra todos os processos remanescentes no cgroup e remove o diretório.
func KillAndDeleteCgroup(name string) error {
	base := "/sys/fs/cgroup" + constants.CgroupPrefix + name

	if err := os.WriteFile(base+"/cgroup.kill", []byte("1"), 0644); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error killing cgroup %s: %w", name, err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		procs, err := os.ReadFile(base + "/cgroup.procs")
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			break
		}
		if len(strings.TrimSpace(string(procs))) == 0 {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("cgroup %s ainda tem processos após cgroup.kill: %s", name, strings.TrimSpace(string(procs)))
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := os.Remove(base); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing cgroup %s: %w", name, err)
	}
	return nil
}
