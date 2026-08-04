package sandbox

import (
	"fmt"
	"strings"
	"time"

	"sige/internal/constants"
)

type Config struct {
	Name          string
	MemoryMB      int64
	CPU           string
	TimeoutSec    int
	TmpLimitMB    int
	MaxFileSizeMB int
	MaxOpenFiles  int
	Stdin         string
}

type ExecutionResult struct {
	Stdout           string        `json:"stdout"`
	Stderr           string        `json:"stderr"`
	Duration         time.Duration `json:"duration"`
	CPUUserTime      time.Duration `json:"cpu_user_time"`
	CPUSystemTime    time.Duration `json:"cpu_system_time"`
	MemoryPeak       int64         `json:"memory_peak_bytes"`
	ExitCode         int           `json:"exit_code"`
	Status           string        `json:"status"`
	LimitMemoryBytes int64         `json:"limit_memory_bytes"`
	LimitCPU         string        `json:"limit_cpu"`
	LimitTimeoutSec  int           `json:"limit_timeout_sec"`
	LimitFileSizeMax int64         `json:"limit_file_size_bytes"`
	LimitOpenFiles   int           `json:"limit_open_files"`
}

func (c Config) GetMemoryBytes() int64 {
	return c.MemoryMB * constants.BytesInMB
}

func (c Config) GetFormattedCPU() string {
	if c.CPU == "" {
		return ""
	}

	if strings.Contains(c.CPU, " ") {
		return c.CPU
	}

	percentStr := strings.TrimSuffix(c.CPU, "%")
	var percent float64
	_, err := fmt.Sscanf(percentStr, "%f", &percent)
	if err != nil {
		return ""
	}

	period := constants.DefaultCPUPeriod
	quota := int(percent * float64(period) / 100.0)
	return fmt.Sprintf("%d %d", quota, period)
}
