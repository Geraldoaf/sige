package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"sige/internal/cgroups"
	"sige/internal/constants"

	"github.com/containerd/cgroups/v3/cgroup2"
)

func Run(mgr *cgroup2.Manager, config Config, cmdStr string, args ...string) (ExecutionResult, error) {
	self := os.Getenv("TCC_EXECUTABLE")
	if self == "" {
		var err error
		self, err = os.Executable()
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("error obtaining executable: %w", err)
		}
	}

	var ctx context.Context
	var cancel context.CancelFunc

	if config.TimeoutSec > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(config.TimeoutSec)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	cgroupPath := constants.CgroupPrefix + config.Name

	rootfsPath, err := os.MkdirTemp("", "tcc-rootfs-")
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error creating temporary rootfs directory: %w", err)
	}
	defer os.RemoveAll(rootfsPath)

	if err := os.Chmod(rootfsPath, constants.RootfsPerms); err != nil {
		return ExecutionResult{}, fmt.Errorf("error changing temporary rootfs permissions: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error obtaining working directory: %w", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error creating pipe: %w", err)
	}
	defer r.Close()

	launchArgs := []string{
		"internal-launch",
		"--sync-fd", "3",
		"--rootfs", rootfsPath,
		"--workspace", wd,
		"--tmp-limit", fmt.Sprintf("%d", config.TmpLimitMB),
		"--file-limit", fmt.Sprintf("%d", config.MaxFileSizeMB),
		"--nofile-limit", fmt.Sprintf("%d", config.MaxOpenFiles),
		"--",
		cmdStr,
	}
	launchArgs = append(launchArgs, args...)

	cmd := exec.CommandContext(ctx, self, launchArgs...)
	cmd.Cancel = func() error {

		killPath := "/sys/fs/cgroup" + cgroupPath + "/cgroup.kill"
		if err := os.WriteFile(killPath, []byte("1"), 0644); err == nil {
			return nil
		}

		if cmd.Process != nil {
			return cmd.Process.Signal(syscall.SIGKILL)
		}
		return nil
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if config.Stdin != "" {
		cmd.Stdin = strings.NewReader(config.Stdin)
	}

	cmd.ExtraFiles = []*os.File{r}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS | syscall.CLONE_NEWIPC,
	}

	if err := cmd.Start(); err != nil {
		w.Close()
		return ExecutionResult{}, fmt.Errorf("error starting process: %w", err)
	}

	startTime := time.Now()

	if err := cgroups.AddProcessToCgroup(cgroupPath, cmd.Process.Pid); err != nil {
		w.Close()
		return ExecutionResult{}, fmt.Errorf("error adding process to cgroup: %w", err)
	}

	w.Close()

	fmt.Printf("Process %d started (timeout: %ds). Waiting...\n", cmd.Process.Pid, config.TimeoutSec)

	err = cmd.Wait()
	duration := time.Since(startTime)

	var memoryPeak int64
	peakPath := "/sys/fs/cgroup" + cgroupPath + "/memory.peak"
	if data, readErr := os.ReadFile(peakPath); readErr == nil {
		if val, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); parseErr == nil {
			memoryPeak = val
		}
	} else {

		currentPath := "/sys/fs/cgroup" + cgroupPath + "/memory.current"
		if data, readErr := os.ReadFile(currentPath); readErr == nil {
			if val, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); parseErr == nil {
				memoryPeak = val
			}
		}
	}

	var exitCode int
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {

			exitCode = -1
		}
	} else {
		exitCode = 0
	}

	var status string = "success"
	if err != nil {
		status = "failed"

		if ctx.Err() == context.DeadlineExceeded {
			status = "timeout"
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			ws := exitErr.Sys().(syscall.WaitStatus)

			if (ws.Signaled() && ws.Signal() == syscall.SIGXFSZ) ||
				strings.Contains(stderrBuf.String(), "File too large") ||
				strings.Contains(stderrBuf.String(), "File size limit exceeded") {
				status = "file_size_exceeded"
			} else {

				eventsPath := "/sys/fs/cgroup" + cgroupPath + "/memory.events"
				if data, oomErr := os.ReadFile(eventsPath); oomErr == nil {
					lines := strings.Split(string(data), "\n")
					for _, line := range lines {
						parts := strings.Fields(line)
						if len(parts) == 2 && parts[0] == "oom_kill" {
							if count, _ := strconv.Atoi(parts[1]); count > 0 {
								status = "oom"
								break
							}
						}
					}
				}

				if status == "failed" && ws.Signaled() && ws.Signal() == syscall.SIGKILL {
					status = "oom"
				}
			}
		}
	}

	var cpuUserTime time.Duration
	var cpuSystemTime time.Duration

	cpuStatPath := "/sys/fs/cgroup" + cgroupPath + "/cpu.stat"
	if data, readErr := os.ReadFile(cpuStatPath); readErr == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				if parts[0] == "user_usec" {
					if usec, parseErr := strconv.ParseInt(parts[1], 10, 64); parseErr == nil {
						cpuUserTime = time.Duration(usec) * time.Microsecond
					}
				} else if parts[0] == "system_usec" {
					if usec, parseErr := strconv.ParseInt(parts[1], 10, 64); parseErr == nil {
						cpuSystemTime = time.Duration(usec) * time.Microsecond
					}
				}
			}
		}
	}
	if cpuUserTime == 0 && cmd.ProcessState != nil {
		cpuUserTime = cmd.ProcessState.UserTime()
	}
	if cpuSystemTime == 0 && cmd.ProcessState != nil {
		cpuSystemTime = cmd.ProcessState.SystemTime()
	}

	res := ExecutionResult{
		Stdout:           stdoutBuf.String(),
		Stderr:           stderrBuf.String(),
		Duration:         duration,
		CPUUserTime:      cpuUserTime,
		CPUSystemTime:    cpuSystemTime,
		MemoryPeak:       memoryPeak,
		ExitCode:         exitCode,
		Status:           status,
		LimitMemoryBytes: config.GetMemoryBytes(),
		LimitCPU:         config.CPU,
		LimitTimeoutSec:  config.TimeoutSec,
		LimitFileSizeMax: int64(config.MaxFileSizeMB) * 1024 * 1024,
		LimitOpenFiles:   config.MaxOpenFiles,
	}

	if ctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("execution time limit exceeded (%ds)", config.TimeoutSec)
	}
	return res, err
}
