package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"sige/internal/constants"
)

type DefaultConfig struct {
	MemoryMB             int64  `json:"memory_mb"`
	CPU                  string `json:"cpu"`
	TimeoutSec           int    `json:"timeout_sec"`
	TmpLimitMB           int    `json:"tmp_limit_mb"`
	MaxFileSizeMB        int    `json:"max_file_size_mb"`
	MaxOpenFiles         int    `json:"max_open_files"`
	APIMode              string `json:"api_mode"`
	CeilingMemoryMB      int64  `json:"ceiling_memory_mb"`
	CeilingCPUPercent    int    `json:"ceiling_cpu_percent"`
	CeilingTimeoutSec    int    `json:"ceiling_timeout_sec"`
	CeilingTmpLimitMB    int    `json:"ceiling_tmp_limit_mb"`
	CeilingMaxFileSizeMB int    `json:"ceiling_max_file_size_mb"`
	CeilingMaxOpenFiles  int    `json:"ceiling_max_open_files"`
}

// GetDefaults retorna as configurações padrão do sistema.
func GetDefaults() DefaultConfig {
	return DefaultConfig{
		MemoryMB:      50,
		CPU:           fmt.Sprintf("%d%%", constants.DefaultFallbackCPUPercent),
		TimeoutSec:    constants.DefaultFallbackTimeoutSec,
		TmpLimitMB:    64,
		MaxFileSizeMB: 15,
		MaxOpenFiles:  256,
		APIMode:       "interpreter",

		CeilingMemoryMB:      constants.DefaultCeilingMemoryMB,
		CeilingCPUPercent:    constants.DefaultCeilingCPUPercent,
		CeilingTimeoutSec:    constants.DefaultCeilingTimeoutSec,
		CeilingTmpLimitMB:    constants.DefaultCeilingTmpLimitMB,
		CeilingMaxFileSizeMB: constants.DefaultCeilingMaxFileSizeMB,
		CeilingMaxOpenFiles:  constants.DefaultCeilingMaxOpenFiles,
	}
}

// Resolve combina configurações padrão, arquivo JSON (se presente) e variáveis de ambiente SIGE_*.
func Resolve(path string) (DefaultConfig, error) {
	cfg := GetDefaults()

	if file, err := os.Open(path); err == nil {
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&cfg); err != nil {
			return cfg, fmt.Errorf("falha no parsing de %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("falha ao abrir %s: %w", path, err)
	}

	if err := applyEnvOverrides(&cfg); err != nil {
		return cfg, err
	}

	return ClampToCeilings(cfg), nil
}

// applyEnvOverrides aplica sobreposições a partir das variáveis de ambiente SIGE_*.
func applyEnvOverrides(cfg *DefaultConfig) error {
	if v, ok := lookupEnv("SIGE_API_MODE"); ok {
		cfg.APIMode = v
	}
	if v, ok := lookupEnv("SIGE_CPU"); ok {
		cfg.CPU = v
	}

	ints := []struct {
		env string
		dst *int
	}{
		{"SIGE_TIMEOUT_SEC", &cfg.TimeoutSec},
		{"SIGE_TMP_LIMIT_MB", &cfg.TmpLimitMB},
		{"SIGE_MAX_FILE_SIZE_MB", &cfg.MaxFileSizeMB},
		{"SIGE_MAX_OPEN_FILES", &cfg.MaxOpenFiles},
		{"SIGE_CEILING_CPU_PERCENT", &cfg.CeilingCPUPercent},
		{"SIGE_CEILING_TIMEOUT_SEC", &cfg.CeilingTimeoutSec},
		{"SIGE_CEILING_TMP_LIMIT_MB", &cfg.CeilingTmpLimitMB},
		{"SIGE_CEILING_MAX_FILE_SIZE_MB", &cfg.CeilingMaxFileSizeMB},
		{"SIGE_CEILING_MAX_OPEN_FILES", &cfg.CeilingMaxOpenFiles},
	}
	for _, f := range ints {
		v, ok := lookupEnv(f.env)
		if !ok {
			continue
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%s inválido (%q): deve ser um número inteiro", f.env, v)
		}
		*f.dst = n
	}

	int64s := []struct {
		env string
		dst *int64
	}{
		{"SIGE_MEMORY_MB", &cfg.MemoryMB},
		{"SIGE_CEILING_MEMORY_MB", &cfg.CeilingMemoryMB},
	}
	for _, f := range int64s {
		v, ok := lookupEnv(f.env)
		if !ok {
			continue
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("%s inválido (%q): deve ser um número inteiro", f.env, v)
		}
		*f.dst = n
	}

	return nil
}

func lookupEnv(name string) (string, bool) {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return "", false
	}
	return v, true
}

type CustomLimits struct {
	MemoryMB      int64
	CPU           string
	TimeoutSec    int
	TmpLimitMB    int
	MaxFileSizeMB int
	MaxOpenFiles  int
}

// MergeLimits mescla as configurações base com os limites customizados da requisição.
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
	return ClampToCeilings(resolved)
}

// ClampToCeilings assegura que nenhum limite ultrapasse os tetos máximos configurados no servidor.
func ClampToCeilings(cfg DefaultConfig) DefaultConfig {
	ceilMem := cfg.CeilingMemoryMB
	if ceilMem <= 0 {
		ceilMem = constants.DefaultCeilingMemoryMB
	}
	if cfg.MemoryMB <= 0 || cfg.MemoryMB > ceilMem {
		cfg.MemoryMB = ceilMem
	}

	ceilTimeout := cfg.CeilingTimeoutSec
	if ceilTimeout <= 0 {
		ceilTimeout = constants.DefaultCeilingTimeoutSec
	}
	if cfg.TimeoutSec <= 0 {
		cfg.TimeoutSec = constants.DefaultFallbackTimeoutSec
	}
	if cfg.TimeoutSec > ceilTimeout {
		cfg.TimeoutSec = ceilTimeout
	}

	ceilTmp := cfg.CeilingTmpLimitMB
	if ceilTmp <= 0 {
		ceilTmp = constants.DefaultCeilingTmpLimitMB
	}
	if cfg.TmpLimitMB <= 0 || cfg.TmpLimitMB > ceilTmp {
		cfg.TmpLimitMB = ceilTmp
	}

	ceilFile := cfg.CeilingMaxFileSizeMB
	if ceilFile <= 0 {
		ceilFile = constants.DefaultCeilingMaxFileSizeMB
	}
	if cfg.MaxFileSizeMB <= 0 || cfg.MaxFileSizeMB > ceilFile {
		cfg.MaxFileSizeMB = ceilFile
	}

	ceilOpenFiles := cfg.CeilingMaxOpenFiles
	if ceilOpenFiles <= 0 {
		ceilOpenFiles = constants.DefaultCeilingMaxOpenFiles
	}
	if cfg.MaxOpenFiles <= 0 || cfg.MaxOpenFiles > ceilOpenFiles {
		cfg.MaxOpenFiles = ceilOpenFiles
	}

	ceilCPU := cfg.CeilingCPUPercent
	if ceilCPU <= 0 {
		ceilCPU = constants.DefaultCeilingCPUPercent
	}
	if percent, ok := cpuPercent(cfg.CPU); !ok {
		cfg.CPU = fmt.Sprintf("%d%%", constants.DefaultFallbackCPUPercent)
	} else if percent > float64(ceilCPU) {
		cfg.CPU = fmt.Sprintf("%d%%", ceilCPU)
	}

	return cfg
}

// cpuPercent converte a especificação de CPU ("10", "10%", ou "quota period") para porcentagem numérica.
func cpuPercent(cpu string) (float64, bool) {
	cpu = strings.TrimSpace(cpu)
	if cpu == "" {
		return 0, false
	}
	if strings.Contains(cpu, " ") {
		parts := strings.Fields(cpu)
		if len(parts) != 2 {
			return 0, false
		}
		quota, errQuota := strconv.ParseFloat(parts[0], 64)
		period, errPeriod := strconv.ParseFloat(parts[1], 64)
		if errQuota != nil || errPeriod != nil || period <= 0 {
			return 0, false
		}
		return quota / period * 100, true
	}
	percent, err := strconv.ParseFloat(strings.TrimSuffix(cpu, "%"), 64)
	if err != nil {
		return 0, false
	}
	return percent, true
}

// FormatBytes converte uma quantidade de bytes em uma string formatada (B, KB, MB, GB, etc.).
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
