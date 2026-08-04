package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type DefaultConfig struct {
	MemoryMB      int64  `json:"memory_mb"`
	CPU           string `json:"cpu"`
	TimeoutSec    int    `json:"timeout_sec"`
	TmpLimitMB    int    `json:"tmp_limit_mb"`
	MaxFileSizeMB int    `json:"max_file_size_mb"`
	MaxOpenFiles  int    `json:"max_open_files"`
	APIMode       string `json:"api_mode"`
}

func GetDefaults() DefaultConfig {
	return DefaultConfig{
		MemoryMB:      50,
		CPU:           "",
		TimeoutSec:    0,
		TmpLimitMB:    64,
		MaxFileSizeMB: 15,
		MaxOpenFiles:  256,
		APIMode:       "interpreter",
	}
}

func LoadConfig(path string) (DefaultConfig, error) {
	var loaded DefaultConfig
	file, err := os.Open(path)
	if err != nil {
		return loaded, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&loaded); err != nil {
		return loaded, fmt.Errorf("falha no parsing do JSON: %w", err)
	}

	return loaded, nil
}

func GenerateDefaultConfig(path string) error {
	defaults := GetDefaults()
	data, err := json.MarshalIndent(defaults, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

type CustomLimits struct {
	MemoryMB      int64
	CPU           string
	TimeoutSec    int
	TmpLimitMB    int
	MaxFileSizeMB int
	MaxOpenFiles  int
}

func MergeLimits(base DefaultConfig, custom CustomLimits) DefaultConfig {
	resolved := base
	if custom.MemoryMB > 0 {
		resolved.MemoryMB = custom.MemoryMB
	}
	if custom.CPU != "" {
		resolved.CPU = custom.CPU
	}
	if custom.TimeoutSec > 0 {
		resolved.TimeoutSec = custom.TimeoutSec
	}
	if custom.TmpLimitMB > 0 {
		resolved.TmpLimitMB = custom.TmpLimitMB
	}
	if custom.MaxFileSizeMB > 0 {
		resolved.MaxFileSizeMB = custom.MaxFileSizeMB
	}
	if custom.MaxOpenFiles > 0 {
		resolved.MaxOpenFiles = custom.MaxOpenFiles
	}
	return resolved
}

func FormatBytes(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
