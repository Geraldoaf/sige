package sandbox

import (
	"os"
	"strings"
	"testing"

	"sige/internal/cgroups"
)

func TestMain(m *testing.M) {

	cgroups.SetupDelegation()
	os.Exit(m.Run())
}

func checkPrivileged(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Pulando teste de integração do sandbox: requer privilégios de root")
	}
	if !cgroups.VerifyCgroupsVersion() {
		t.Skip("Pulando teste de integração do sandbox: cgroup v2 unificado não disponível")
	}
}

func TestExecuteEcho(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-echo-int",
		MemoryMB:   50,
		TimeoutSec: 5,
	}

	result, err := Execute(cfg, "/bin/echo", []string{"Hello Integration Test"})
	if err != nil {
		t.Fatalf("Execute falhou: %v", err)
	}

	if result.Duration <= 0 {
		t.Errorf("Duração inválida detectada para o teste: %v", result.Duration)
	}
}

func TestExecuteTimeout(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-timeout-int",
		MemoryMB:   50,
		TimeoutSec: 1,
	}

	result, err := Execute(cfg, "/bin/sleep", []string{"3"})
	if err == nil {
		t.Fatal("Esperava falha por estouro de tempo (timeout), mas o comando executou com sucesso")
	}

	if result.Status != "timeout" {
		t.Errorf("Esperava status 'timeout', obteve: %q", result.Status)
	}

	if !strings.Contains(err.Error(), "execution time limit exceeded") {
		t.Errorf("Expected error containing 'execution time limit exceeded', got: %v", err)
	}
}

func TestExecuteSeccompBlock(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-seccomp-int",
		MemoryMB:   50,
		TimeoutSec: 5,
	}

	_, err := Execute(cfg, "/usr/bin/unshare", []string{"--user"})
	if err == nil {
		t.Fatal("Esperava que o comando fosse bloqueado pelo Seccomp, mas ele executou com sucesso")
	}

	if !strings.Contains(err.Error(), "bad system call") {
		t.Errorf("Esperava erro 'bad system call' do Seccomp, obteve: %v", err)
	}
}

func TestExecuteWorkspaceReadOnly(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-ro-workspace",
		MemoryMB:   50,
		TimeoutSec: 5,
	}

	_, err := Execute(cfg, "/usr/bin/touch", []string{"/workspace/test_ro_file.txt"})
	if err == nil {
		t.Fatal("Esperava que a escrita no workspace falhasse por ser Read-Only, mas o comando executou com sucesso")
	}
}

func TestExecuteEnvironmentSanitized(t *testing.T) {
	checkPrivileged(t)

	os.Setenv("TEST_HOST_ENV", "leaked_secret")
	defer os.Unsetenv("TEST_HOST_ENV")

	cfg := Config{
		Name:       "test-clean-env",
		MemoryMB:   50,
		TimeoutSec: 5,
	}

	_, err := Execute(cfg, "/bin/sh", []string{"-c", `if [ -n "$TEST_HOST_ENV" ]; then exit 42; fi`})
	if err != nil {
		t.Fatalf("O ambiente não foi sanitizado, variável de host vazou para o sandbox: %v", err)
	}
}

func TestExecuteTmpfsReadOnly(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-tmpfs-ro",
		MemoryMB:   50,
		TimeoutSec: 5,
		TmpLimitMB: 0,
	}

	_, err := Execute(cfg, "/usr/bin/touch", []string{"/tmp/test_ro_file.txt"})
	if err == nil {
		t.Fatal("Esperava que a escrita no /tmp falhasse por ser Read-Only, mas o comando executou com sucesso")
	}
}

func TestExecuteTmpfsCustomLimit(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-tmpfs-custom-limit",
		MemoryMB:   100,
		TimeoutSec: 5,
		TmpLimitMB: 10,
	}

	_, err := Execute(cfg, "/usr/bin/python3", []string{"-c", "open('/tmp/bigfile', 'wb').write(b'\\x00' * 12 * 1024 * 1024)"})
	if err == nil {
		t.Fatal("Esperava que a escrita de 12MB no /tmp falhasse por limite do tmpfs (10MB), mas o comando executou com sucesso")
	}
}

func TestExecuteInvalidSandboxName(t *testing.T) {
	cfg := Config{
		Name:       "../invalid/path",
		MemoryMB:   50,
		TimeoutSec: 5,
	}

	_, err := Execute(cfg, "/bin/echo", []string{"fail"})
	if err == nil {
		t.Fatal("Esperava erro de validação para nome de sandbox inválido, mas a execução retornou sucesso")
	}

	if !strings.Contains(err.Error(), "invalid sandbox name") {
		t.Errorf("Expected error containing 'invalid sandbox name', got: %v", err)
	}
}

func TestExecuteFileLimitExceeded(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:          "test-file-limit-exceeded",
		MemoryMB:      100,
		TimeoutSec:    5,
		TmpLimitMB:    64,
		MaxFileSizeMB: 5,
	}

	result, err := Execute(cfg, "/usr/bin/python3", []string{"-c", "open('/tmp/large_file', 'wb').write(b'\\x00' * 8 * 1024 * 1024)"})
	if err == nil {
		t.Fatal("Esperava que a escrita de 8MB falhasse por limite de arquivo (5MB), mas o comando executou com sucesso")
	}

	if result.Status != "file_size_exceeded" {
		t.Errorf("Esperava status 'file_size_exceeded', obteve: %q", result.Status)
	}
}

func TestExecuteFileLimitDefault(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:          "test-file-limit-default",
		MemoryMB:      100,
		TimeoutSec:    5,
		TmpLimitMB:    64,
		MaxFileSizeMB: 15,
	}

	_, err := Execute(cfg, "/usr/bin/python3", []string{"-c", "open('/tmp/large_file', 'wb').write(b'\\x00' * 10 * 1024 * 1024)"})
	if err != nil {
		t.Fatalf("Erro inesperado ao escrever arquivo de 10MB dentro do limite (15MB): %v", err)
	}
}

func TestExecuteCaptureOutput(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-capture-output",
		MemoryMB:   100,
		TimeoutSec: 5,
	}

	pyScript := `import sys; print("Stdout message"); print("Stderr traceback", file=sys.stderr)`

	result, err := Execute(cfg, "/usr/bin/python3", []string{"-c", pyScript})
	if err != nil {
		t.Fatalf("Erro inesperado ao executar: %v", err)
	}

	if !strings.Contains(result.Stdout, "Stdout message") {
		t.Errorf("Esperava capturar 'Stdout message' no stdout, obteve: %q", result.Stdout)
	}

	if !strings.Contains(result.Stderr, "Stderr traceback") {
		t.Errorf("Esperava capturar 'Stderr traceback' no stderr, obteve: %q", result.Stderr)
	}
}

func TestExecuteMemoryMeasurement(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-memory-measurement",
		MemoryMB:   100,
		TimeoutSec: 5,
	}

	pyScript := `x = b'\x00' * 12 * 1024 * 1024; print("Allocated memory successfully")`

	result, err := Execute(cfg, "/usr/bin/python3", []string{"-c", pyScript})
	if err != nil {
		t.Fatalf("Erro inesperado ao executar: %v", err)
	}

	if result.MemoryPeak < 8*1024*1024 {
		t.Errorf("Esperava que o pico de memória medido fosse maior que 8MB (8388608 bytes), obtido: %d bytes", result.MemoryPeak)
	}
}

func TestExecuteOOMKilled(t *testing.T) {
	checkPrivileged(t)

	cfg := Config{
		Name:       "test-oom-killed",
		MemoryMB:   15,
		TimeoutSec: 5,
	}

	pyScript := `x = b'\x00' * 45 * 1024 * 1024; print("Allocated memory successfully")`

	result, err := Execute(cfg, "/usr/bin/python3", []string{"-c", pyScript})
	if err == nil {
		t.Fatal("Esperava que a execução falhasse por estouro de memória (OOM), mas o comando executou com sucesso")
	}

	if result.Status != "oom" {
		t.Errorf("Esperava status 'oom', obteve: %q (erro: %v)", result.Status, err)
	}
}
