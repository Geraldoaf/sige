package config

import (
	"path/filepath"
	"testing"
)

func TestCpuPercent_VariousFormats(t *testing.T) {
	tests := []struct {
		input       string
		expectedVal float64
		expectedOk  bool
	}{
		{"", 0, false},
		{"   ", 0, false},
		{"invalid", 0, false},
		{"50%", 50.0, true},
		{"100%", 100.0, true},
		{"25.5%", 25.5, true},
		{"50", 50.0, true},
		{"10000 100000", 10.0, true},
		{"50000 100000", 50.0, true},
		{"100000 100000", 100.0, true},
		{"10000 0", 0, false},    // division by zero in period
		{"10000 -5", 0, false},   // negative period
		{"10 20 30", 0, false},   // too many parts
		{"abc 100000", 0, false}, // invalid quota
		{"10000 abc", 0, false},  // invalid period
	}

	for _, tt := range tests {
		val, ok := cpuPercent(tt.input)
		if ok != tt.expectedOk {
			t.Errorf("cpuPercent(%q) ok = %v, expected %v", tt.input, ok, tt.expectedOk)
		}
		if ok && val != tt.expectedVal {
			t.Errorf("cpuPercent(%q) val = %f, expected %f", tt.input, val, tt.expectedVal)
		}
	}
}

func TestFormatBytes_EdgeCases(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}

	for _, tt := range tests {
		got := FormatBytes(tt.bytes)
		if got != tt.expected {
			t.Errorf("FormatBytes(%d) = %q, expected %q", tt.bytes, got, tt.expected)
		}
	}
}

func TestResolve_AllEnvOverrides(t *testing.T) {
	t.Setenv("SIGE_API_MODE", "single_evaluation")
	t.Setenv("SIGE_CPU", "25%")
	t.Setenv("SIGE_TIMEOUT_SEC", "15")
	t.Setenv("SIGE_TMP_LIMIT_MB", "128")
	t.Setenv("SIGE_MAX_FILE_SIZE_MB", "30")
	t.Setenv("SIGE_MAX_OPEN_FILES", "100")
	t.Setenv("SIGE_MEMORY_MB", "200")
	t.Setenv("SIGE_CEILING_MEMORY_MB", "1024")
	t.Setenv("SIGE_CEILING_CPU_PERCENT", "200")
	t.Setenv("SIGE_CEILING_TIMEOUT_SEC", "60")
	t.Setenv("SIGE_CEILING_TMP_LIMIT_MB", "512")
	t.Setenv("SIGE_CEILING_MAX_FILE_SIZE_MB", "128")
	t.Setenv("SIGE_CEILING_MAX_OPEN_FILES", "1024")

	cfg, err := Resolve(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if cfg.APIMode != "single_evaluation" {
		t.Errorf("APIMode = %q, expected 'single_evaluation'", cfg.APIMode)
	}
	if cfg.CPU != "25%" {
		t.Errorf("CPU = %q, expected '25%%'", cfg.CPU)
	}
	if cfg.TimeoutSec != 15 {
		t.Errorf("TimeoutSec = %d, expected 15", cfg.TimeoutSec)
	}
	if cfg.TmpLimitMB != 128 {
		t.Errorf("TmpLimitMB = %d, expected 128", cfg.TmpLimitMB)
	}
	if cfg.MaxFileSizeMB != 30 {
		t.Errorf("MaxFileSizeMB = %d, expected 30", cfg.MaxFileSizeMB)
	}
	if cfg.MaxOpenFiles != 100 {
		t.Errorf("MaxOpenFiles = %d, expected 100", cfg.MaxOpenFiles)
	}
	if cfg.MemoryMB != 200 {
		t.Errorf("MemoryMB = %d, expected 200", cfg.MemoryMB)
	}
}

func TestResolve_InvalidIntEnv(t *testing.T) {
	invalidEnvs := []string{
		"SIGE_TIMEOUT_SEC",
		"SIGE_TMP_LIMIT_MB",
		"SIGE_MAX_FILE_SIZE_MB",
		"SIGE_MAX_OPEN_FILES",
		"SIGE_CEILING_CPU_PERCENT",
		"SIGE_CEILING_TIMEOUT_SEC",
		"SIGE_CEILING_TMP_LIMIT_MB",
		"SIGE_CEILING_MAX_FILE_SIZE_MB",
		"SIGE_CEILING_MAX_OPEN_FILES",
		"SIGE_MEMORY_MB",
		"SIGE_CEILING_MEMORY_MB",
	}

	for _, env := range invalidEnvs {
		t.Run(env, func(t *testing.T) {
			t.Setenv(env, "not-a-number")
			_, err := Resolve(filepath.Join(t.TempDir(), "nonexistent.json"))
			if err == nil {
				t.Errorf("expected error when %s is not a number, got nil", env)
			}
		})
	}
}
