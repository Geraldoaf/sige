package cgroups

import (
	"fmt"
	"os"

	"sige/internal/constants"

	"github.com/containerd/cgroups/v3/cgroup2"
)

func CreateCgroup(name string, memLimitBytes int64, cpuMax string) (*cgroup2.Manager, error) {
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

func JoinCgroup(path string) error {
	return AddProcessToCgroup(path, 0)
}

func AddProcessToCgroup(path string, pid int) error {
	data := []byte(fmt.Sprintf("%d", pid))
	return os.WriteFile("/sys/fs/cgroup"+path+"/cgroup.procs", data, 0644)
}

func DeleteCgroup(mgr *cgroup2.Manager) error {
	return mgr.Delete()
}
