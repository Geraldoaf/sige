package sandbox_test

import (
	"os"
	"path/filepath"
	"sige/internal/sandbox"
	"testing"
)

func TestIsSafeFilename(t *testing.T) {
	valid := []string{"main.py", "solution.c", "test-1_2.cpp", "script.sh"}
	for _, f := range valid {
		if !sandbox.IsSafeFilename(f) {
			t.Errorf("esperado nome valido para %q", f)
		}
	}

	invalid := []string{".", "..", "../evil.py", "dir/file.py", "test;rm.py", "space name.py"}
	for _, f := range invalid {
		if sandbox.IsSafeFilename(f) {
			t.Errorf("esperado nome invalido para %q", f)
		}
	}
}

func TestResolveDefaultFilename(t *testing.T) {
	if f := sandbox.ResolveDefaultFilename("python", ""); f != "main.py" {
		t.Errorf("esperado main.py, obtido %s", f)
	}
	if f := sandbox.ResolveDefaultFilename("c", ""); f != "solution.c" {
		t.Errorf("esperado solution.c, obtido %s", f)
	}
	if f := sandbox.ResolveDefaultFilename("python", "custom.py"); f != "custom.py" {
		t.Errorf("esperado custom.py, obtido %s", f)
	}
}

func TestPrepareWorkspace_PythonScript(t *testing.T) {
	tempBase := t.TempDir()
	spec := sandbox.WorkspaceSpec{
		Language: "python",
		Code:     "print('teste')",
		BaseDir:  tempBase,
		ExecID:   "exec-test-1",
	}

	prep, err := sandbox.PrepareWorkspace(spec)
	if err != nil {
		t.Fatalf("erro ao preparar workspace: %v", err)
	}
	defer prep.Cleanup()

	if prep.Command != "/usr/bin/python3" {
		t.Errorf("esperado comando /usr/bin/python3, obtido %s", prep.Command)
	}

	// Verifica se o arquivo foi materializado
	expectedFile := filepath.Join(prep.HostWd, "main.py")
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("erro ao ler arquivo criado: %v", err)
	}
	if string(content) != "print('teste')" {
		t.Errorf("conteudo incorreto: %s", string(content))
	}
}
