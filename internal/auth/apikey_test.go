package auth_test

import (
	"os"
	"path/filepath"
	"sige/internal/auth"
	"testing"
)

func TestFileKeyProvider_EnvKey(t *testing.T) {
	os.Setenv("SIGE_API_KEY", "test-secret-key-123")
	defer os.Unsetenv("SIGE_API_KEY")

	provider := auth.NewFileKeyProvider(t.TempDir(), false)
	key := provider.ResolveKey()

	if key != "test-secret-key-123" {
		t.Fatalf("esperado test-secret-key-123, obtido %s", key)
	}
	if !provider.Validate("test-secret-key-123") {
		t.Errorf("Validate deveria retornar true para a chave correta")
	}
	if provider.Validate("wrong-key") {
		t.Errorf("Validate deveria retornar false para chave incorreta")
	}
}

func TestFileKeyProvider_GeneratesAndPersistsKey(t *testing.T) {
	os.Unsetenv("SIGE_API_KEY")
	os.Unsetenv("SIGE_ALLOW_UNAUTHENTICATED")

	tempDir := t.TempDir()
	provider := auth.NewFileKeyProvider(tempDir, false)

	key1 := provider.ResolveKey()
	if len(key1) != 64 { // 32 bytes em hex
		t.Fatalf("esperado chave de 64 caracteres hex, obtido %d (%s)", len(key1), key1)
	}
	if !provider.IsGenerated() {
		t.Errorf("esperado IsGenerated=true para primeira geração")
	}

	// Segundo provider apontando para o mesmo diretório deve carregar a chave persistida
	provider2 := auth.NewFileKeyProvider(tempDir, false)
	key2 := provider2.ResolveKey()

	if key2 != key1 {
		t.Fatalf("esperado reutilizar a chave persistida (%s), obtido nova chave (%s)", key1, key2)
	}
	if provider2.IsGenerated() {
		t.Errorf("esperado IsGenerated=false ao recarregar chave do disco")
	}

	// Valida permissão do arquivo
	info, err := os.Stat(filepath.Join(tempDir, "api_key"))
	if err != nil {
		t.Fatalf("arquivo api_key nao encontrado: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissao esperada 0600, obtida %v", info.Mode().Perm())
	}
}

func TestFileKeyProvider_AllowUnauthenticated(t *testing.T) {
	os.Unsetenv("SIGE_API_KEY")
	provider := auth.NewFileKeyProvider(t.TempDir(), true)

	if !provider.Validate("") {
		t.Errorf("Validate deveria retornar true quando allowUnauthenticated=true")
	}
}
