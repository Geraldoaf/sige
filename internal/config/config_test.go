package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMissing(t *testing.T) {
	_, err := LoadConfig("non_existent_file.json")
	if err == nil {
		t.Fatal("Esperava erro ao tentar carregar arquivo inexistente, obteve nil")
	}
}

func TestGenerateAndLoadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sige-config-test-")
	if err != nil {
		t.Fatalf("Erro ao criar diretório temporário: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	err = GenerateDefaultConfig(configPath)
	if err != nil {
		t.Fatalf("Erro ao gerar configuração padrão: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Erro ao ler configuração gerada: %v", err)
	}

	defaults := GetDefaults()

	if loaded.MemoryMB != defaults.MemoryMB {
		t.Errorf("Esperava MemoryMB %d, obteve %d", defaults.MemoryMB, loaded.MemoryMB)
	}
	if loaded.TmpLimitMB != defaults.TmpLimitMB {
		t.Errorf("Esperava TmpLimitMB %d, obteve %d", defaults.TmpLimitMB, loaded.TmpLimitMB)
	}
	if loaded.MaxFileSizeMB != defaults.MaxFileSizeMB {
		t.Errorf("Esperava MaxFileSizeMB %d, obteve %d", defaults.MaxFileSizeMB, loaded.MaxFileSizeMB)
	}
	if loaded.MaxOpenFiles != defaults.MaxOpenFiles {
		t.Errorf("Esperava MaxOpenFiles %d, obteve %d", defaults.MaxOpenFiles, loaded.MaxOpenFiles)
	}
	if loaded.APIMode != defaults.APIMode {
		t.Errorf("Esperava APIMode %q, obteve %q", defaults.APIMode, loaded.APIMode)
	}
}

func TestMergeLimits(t *testing.T) {
	base := DefaultConfig{
		MemoryMB:      50,
		CPU:           "10",
		TimeoutSec:    5,
		TmpLimitMB:    64,
		MaxFileSizeMB: 15,
		MaxOpenFiles:  256,
	}

	custom := CustomLimits{
		MemoryMB:   100,
		TimeoutSec: 10,
	}

	merged := MergeLimits(base, custom)

	if merged.MemoryMB != 100 {
		t.Errorf("Esperava MemoryMB 100, obteve %d", merged.MemoryMB)
	}
	if merged.TimeoutSec != 10 {
		t.Errorf("Esperava TimeoutSec 10, obteve %d", merged.TimeoutSec)
	}
	if merged.CPU != "10" {
		t.Errorf("Esperava CPU '10', obteve %q", merged.CPU)
	}
	if merged.TmpLimitMB != 64 {
		t.Errorf("Esperava TmpLimitMB 64, obteve %d", merged.TmpLimitMB)
	}
}

func TestFormatBytes(t *testing.T) {
	if got := FormatBytes(500); got != "500 B" {
		t.Errorf("Esperava '500 B', obteve %q", got)
	}
	if got := FormatBytes(1500); got != "1.46 KB" {
		t.Errorf("Esperava '1.46 KB', obteve %q", got)
	}
	if got := FormatBytes(2 * 1024 * 1024); got != "2.00 MB" {
		t.Errorf("Esperava '2.00 MB', obteve %q", got)
	}
}
