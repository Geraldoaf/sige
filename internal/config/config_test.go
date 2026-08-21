package config

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestMergeLimitsClampsAboveCeiling(t *testing.T) {
	base := DefaultConfig{
		MemoryMB:      50,
		CPU:           "10",
		TimeoutSec:    5,
		TmpLimitMB:    64,
		MaxFileSizeMB: 15,
		MaxOpenFiles:  256,

		CeilingMemoryMB:      512,
		CeilingCPUPercent:    100,
		CeilingTimeoutSec:    30,
		CeilingTmpLimitMB:    256,
		CeilingMaxFileSizeMB: 64,
		CeilingMaxOpenFiles:  512,
	}

	custom := CustomLimits{
		MemoryMB:      1_000_000,
		CPU:           "500%",
		TimeoutSec:    3_600_000,
		TmpLimitMB:    500_000,
		MaxFileSizeMB: 100_000,
		MaxOpenFiles:  1_000_000,
	}

	merged := MergeLimits(base, custom)

	if merged.MemoryMB != base.CeilingMemoryMB {
		t.Errorf("Esperava MemoryMB clampado em %d, obteve %d", base.CeilingMemoryMB, merged.MemoryMB)
	}
	if merged.TimeoutSec != base.CeilingTimeoutSec {
		t.Errorf("Esperava TimeoutSec clampado em %d, obteve %d", base.CeilingTimeoutSec, merged.TimeoutSec)
	}
	if merged.TmpLimitMB != base.CeilingTmpLimitMB {
		t.Errorf("Esperava TmpLimitMB clampado em %d, obteve %d", base.CeilingTmpLimitMB, merged.TmpLimitMB)
	}
	if merged.MaxFileSizeMB != base.CeilingMaxFileSizeMB {
		t.Errorf("Esperava MaxFileSizeMB clampado em %d, obteve %d", base.CeilingMaxFileSizeMB, merged.MaxFileSizeMB)
	}
	if merged.MaxOpenFiles != base.CeilingMaxOpenFiles {
		t.Errorf("Esperava MaxOpenFiles clampado em %d, obteve %d", base.CeilingMaxOpenFiles, merged.MaxOpenFiles)
	}
	if merged.CPU != "100%" {
		t.Errorf("Esperava CPU clampado em '100%%', obteve %q", merged.CPU)
	}
}

func TestClampToCeilingsRejectsUnlimitedDefaults(t *testing.T) {
	// Simula um config.json antigo (pré-teto): timeout_sec=0 e cpu="" eram
	// interpretados como "sem limite" — depois do clamp, nunca mais devem sair
	// como ausentes/zerados.
	cfg := DefaultConfig{
		MemoryMB:      50,
		CPU:           "",
		TimeoutSec:    0,
		TmpLimitMB:    64,
		MaxFileSizeMB: 15,
		MaxOpenFiles:  256,
	}

	clamped := ClampToCeilings(cfg)

	if clamped.TimeoutSec <= 0 {
		t.Errorf("Esperava TimeoutSec > 0 após o clamp, obteve %d", clamped.TimeoutSec)
	}
	if clamped.CPU == "" {
		t.Error("Esperava CPU não-vazio (limitado) após o clamp, obteve string vazia")
	}
}

func TestClampToCeilingsUsesFallbackWhenUnconfigured(t *testing.T) {
	// Ceiling_* zerados (config.json sem esses campos) devem cair nos tetos
	// embutidos no binário, não em "sem teto".
	cfg := DefaultConfig{
		MemoryMB:      10_000_000,
		CPU:           "1000%",
		TimeoutSec:    999_999,
		TmpLimitMB:    10_000_000,
		MaxFileSizeMB: 10_000_000,
		MaxOpenFiles:  10_000_000,
	}

	clamped := ClampToCeilings(cfg)

	if clamped.MemoryMB != 512 {
		t.Errorf("Esperava MemoryMB clampado no fallback (512), obteve %d", clamped.MemoryMB)
	}
	if clamped.TimeoutSec != 30 {
		t.Errorf("Esperava TimeoutSec clampado no fallback (30), obteve %d", clamped.TimeoutSec)
	}
	if clamped.CPU != "100%" {
		t.Errorf("Esperava CPU clampado no fallback ('100%%'), obteve %q", clamped.CPU)
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

func TestResolveSemArquivoUsaPadroes(t *testing.T) {
	cfg, err := Resolve(filepath.Join(t.TempDir(), "inexistente.json"))
	if err != nil {
		t.Fatalf("Resolve deveria tratar arquivo ausente como opcional, obteve: %v", err)
	}
	defaults := GetDefaults()
	if cfg.MemoryMB != defaults.MemoryMB || cfg.APIMode != defaults.APIMode {
		t.Errorf("Esperava os padrões embutidos, obteve MemoryMB=%d APIMode=%q", cfg.MemoryMB, cfg.APIMode)
	}
}

func TestResolveArquivoParcialPreservaPadroes(t *testing.T) {
	// JSON com apenas um campo: os demais NÃO podem virar zero.
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"memory_mb": 128}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Resolve(path)
	if err != nil {
		t.Fatalf("Resolve falhou: %v", err)
	}
	if cfg.MemoryMB != 128 {
		t.Errorf("Esperava MemoryMB 128 do arquivo, obteve %d", cfg.MemoryMB)
	}
	if cfg.TimeoutSec != GetDefaults().TimeoutSec {
		t.Errorf("Campo ausente no JSON zerou o padrão: TimeoutSec=%d", cfg.TimeoutSec)
	}
}

func TestResolveEnvTemPrecedenciaSobreArquivo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"memory_mb": 128, "api_mode": "interpreter"}`), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SIGE_MEMORY_MB", "256")
	t.Setenv("SIGE_API_MODE", "multi_evaluation")

	cfg, err := Resolve(path)
	if err != nil {
		t.Fatalf("Resolve falhou: %v", err)
	}
	if cfg.MemoryMB != 256 {
		t.Errorf("Esperava env sobrepondo o arquivo (256), obteve %d", cfg.MemoryMB)
	}
	if cfg.APIMode != "multi_evaluation" {
		t.Errorf("Esperava APIMode do env, obteve %q", cfg.APIMode)
	}
}

func TestResolveAplicaTetosSobreEnv(t *testing.T) {
	t.Setenv("SIGE_MEMORY_MB", "99999")

	cfg, err := Resolve(filepath.Join(t.TempDir(), "inexistente.json"))
	if err != nil {
		t.Fatalf("Resolve falhou: %v", err)
	}
	if cfg.MemoryMB != GetDefaults().CeilingMemoryMB {
		t.Errorf("Env acima do teto deveria ser limitada a %d, obteve %d",
			GetDefaults().CeilingMemoryMB, cfg.MemoryMB)
	}
}

func TestResolveEnvInvalidaRetornaErro(t *testing.T) {
	t.Setenv("SIGE_TIMEOUT_SEC", "nao-e-numero")

	if _, err := Resolve(filepath.Join(t.TempDir(), "inexistente.json")); err == nil {
		t.Fatal("Esperava erro para SIGE_TIMEOUT_SEC malformada, obteve nil")
	}
}
