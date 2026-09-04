package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"sige/internal/cgroups"
	"sige/internal/constants"

	"github.com/containerd/cgroups/v3/cgroup2"
)

// limitedBuffer limita o tamanho do buffer de saída em memória.
type limitedBuffer struct {
	mu       sync.Mutex
	buf      bytes.Buffer
	limit    int
	exceeded bool
	onExceed func()
}

func newLimitedBuffer(limit int, onExceed func()) *limitedBuffer {
	return &limitedBuffer{limit: limit, onExceed: onExceed}
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	remaining := b.limit - b.buf.Len()
	justExceeded := false
	if remaining > 0 {
		toWrite := p
		if len(toWrite) > remaining {
			toWrite = toWrite[:remaining]
		}
		b.buf.Write(toWrite)
	}
	if b.buf.Len() >= b.limit && !b.exceeded {
		b.exceeded = true
		justExceeded = true
	}
	b.mu.Unlock()

	if justExceeded && b.onExceed != nil {
		b.onExceed()
	}

	return len(p), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func (b *limitedBuffer) Exceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exceeded
}

// Run executa o comando no sandbox associado ao cgroups e monitora limites de execução.
func Run(mgr *cgroup2.Manager, config Config, cmdStr string, args ...string) (ExecutionResult, error) {
	self := os.Getenv("SIGE_EXECUTABLE")
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

	rootfsPath, err := os.MkdirTemp("", "sige-rootfs-")
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("error creating temporary rootfs directory: %w", err)
	}
	defer os.RemoveAll(rootfsPath)

	if err := os.Chmod(rootfsPath, constants.RootfsPerms); err != nil {
		return ExecutionResult{}, fmt.Errorf("error changing temporary rootfs permissions: %w", err)
	}

	wd := config.Workspace
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("error obtaining working directory: %w", err)
		}
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
	}
	if config.WorkspaceWritable {
		launchArgs = append(launchArgs, "--workspace-writable")
	}
	launchArgs = append(launchArgs, "--", cmdStr)
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

	var outputLimitHit int32
	onOutputLimitExceeded := func() {
		if atomic.CompareAndSwapInt32(&outputLimitHit, 0, 1) {
			cancel()
		}
	}
	stdoutBuf := newLimitedBuffer(constants.DefaultMaxOutputBytes, onOutputLimitExceeded)
	stderrBuf := newLimitedBuffer(constants.DefaultMaxOutputBytes, onOutputLimitExceeded)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	if config.Stdin != "" {
		cmd.Stdin = strings.NewReader(config.Stdin)
	}

	cmd.ExtraFiles = []*os.File{r}

	if err := cmd.Start(); err != nil {
		w.Close()
		return ExecutionResult{}, fmt.Errorf("error starting process: %w", err)
	}

	startTime := time.Now()

	if err := cgroups.AddProcessToCgroup(cgroupPath, cmd.Process.Pid); err != nil {
		w.Close()
		_ = cmd.Process.Kill()
		return ExecutionResult{}, fmt.Errorf("error adding process to cgroup: %w", err)
	}

	w.Close()

	fmt.Fprintf(os.Stderr, "[SIGE] pid=%d timeout=%ds aguardando...\n", cmd.Process.Pid, config.TimeoutSec)

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
	if atomic.LoadInt32(&outputLimitHit) == 1 {
		// Prioridade sobre o resto: independente de err ser nil (corrida em
		// que o processo termina "sozinho" bem perto do momento do kill),
		// se o limite de output foi atingido a execução é tratada como tal.
		status = "output_limit_exceeded"
	} else if err != nil {
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

	if atomic.LoadInt32(&outputLimitHit) == 1 {
		return res, fmt.Errorf("output limit exceeded (%d bytes)", constants.DefaultMaxOutputBytes)
	}
	if ctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("execution time limit exceeded (%ds)", config.TimeoutSec)
	}
	return res, err
}
