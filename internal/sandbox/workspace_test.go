package sandbox_test

import (
	"encoding/base64"
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

	invalid := []string{".", "..", "../evil.py", "dir/file.py", "test;rm.py", "space name.py", "-flag.py", ""}
	for _, f := range invalid {
		if sandbox.IsSafeFilename(f) {
			t.Errorf("esperado nome invalido para %q", f)
		}
	}
}

func TestIsSafeRelativePath(t *testing.T) {
	valid := []string{
		"main.py",
		"solution.c",
		"include/calc.h",
		"src/utils/helper.c",
		"data/input_1.txt",
	}
	for _, path := range valid {
		if !sandbox.IsSafeRelativePath(path) {
			t.Errorf("esperado caminho valido para %q", path)
		}
	}

	invalid := []string{
		"",
		".",
		"..",
		"../main.py",
		"../../etc/passwd",
		"/etc/passwd",
		"\\windows\\path",
		"src/../main.c",
		"src/./main.c",
		"src//main.c",
		"src/dir/",
		"-flag.c",
		"src/-flag.c",
		"a/b/c/d/e/f/g/h/i/j/k/too_deep.c",
		"main file.py",
		"calc;rm.c",
	}
	for _, path := range invalid {
		if sandbox.IsSafeRelativePath(path) {
			t.Errorf("esperado caminho invalido para %q", path)
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

func TestPrepareWorkspace_MultiFiles(t *testing.T) {
	tempBase := t.TempDir()
	helperB64 := base64.StdEncoding.EncodeToString([]byte("def get_value(): return 42\n"))

	spec := sandbox.WorkspaceSpec{
		Language: "python",
		Files: []sandbox.FileEntry{
			{
				Name:    "main.py",
				Content: "from lib.helper import get_value\nprint(get_value())",
			},
			{
				Name:          "lib/helper.py",
				ContentBase64: helperB64,
			},
		},
		BaseDir: tempBase,
		ExecID:  "exec-multi-test",
	}

	prep, err := sandbox.PrepareWorkspace(spec)
	if err != nil {
		t.Fatalf("erro ao preparar workspace com multiplos arquivos: %v", err)
	}
	defer prep.Cleanup()

	if prep.Command != "/usr/bin/python3" {
		t.Errorf("esperado comando /usr/bin/python3, obtido %s", prep.Command)
	}

	// Verifica permissões 0755 para que o usuário nobody (others) tenha r-x (0005),
	// indispensável para os.listdir() usado pelo importlib do Python
	for _, dirPath := range []string{prep.HostWd, filepath.Join(prep.HostWd, "lib")} {
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("erro no stat de %s: %v", dirPath, err)
		}
		if info.Mode().Perm()&0005 != 0005 {
			t.Errorf("diretorio %s sem permissao r-x para others (nobody): modo=%04o", dirPath, info.Mode().Perm())
		}
	}

	// Verifica arquivo main.py
	mainFile := filepath.Join(prep.HostWd, "main.py")
	mainContent, err := os.ReadFile(mainFile)
	if err != nil {
		t.Fatalf("erro ao ler main.py: %v", err)
	}
	if string(mainContent) != "from lib.helper import get_value\nprint(get_value())" {
		t.Errorf("conteudo incorreto em main.py: %s", string(mainContent))
	}

	// Verifica arquivo lib/helper.py decodificado do base64
	helperFile := filepath.Join(prep.HostWd, "lib", "helper.py")
	helperContent, err := os.ReadFile(helperFile)
	if err != nil {
		t.Fatalf("erro ao ler lib/helper.py: %v", err)
	}
	if string(helperContent) != "def get_value(): return 42\n" {
		t.Errorf("conteudo incorreto em lib/helper.py: %s", string(helperContent))
	}
}

func TestValidateFileEntries_EdgeCasesAndCollisions(t *testing.T) {
	tests := []struct {
		name        string
		language    string
		files       []sandbox.FileEntry
		entry       string
		expectError bool
	}{
		{
			name:     "Duplicate filenames",
			language: "c",
			files: []sandbox.FileEntry{
				{Name: "main.c", Content: "int main(){return 0;}"},
				{Name: "main.c", Content: "int main(){return 1;}"},
			},
			expectError: true,
		},
		{
			name:     "File collides with directory (file first, then dir)",
			language: "python",
			files: []sandbox.FileEntry{
				{Name: "lib", Content: "x = 1"},
				{Name: "lib/helper.py", Content: "y = 2"},
			},
			expectError: true,
		},
		{
			name:     "Directory collides with file (dir first, then file)",
			language: "python",
			files: []sandbox.FileEntry{
				{Name: "lib/helper.py", Content: "y = 2"},
				{Name: "lib", Content: "x = 1"},
			},
			expectError: true,
		},
		{
			name:     "Reserved binary name 'solution' in C",
			language: "c",
			files: []sandbox.FileEntry{
				{Name: "solution", Content: "fake binary"},
				{Name: "main.c", Content: "int main(){return 0;}"},
			},
			expectError: true,
		},
		{
			name:     "Reserved directory name 'solution/main.c' in C++",
			language: "cpp",
			files: []sandbox.FileEntry{
				{Name: "solution/main.cpp", Content: "int main(){return 0;}"},
			},
			expectError: true,
		},
		{
			name:     "Header-only submission in C without .c source",
			language: "c",
			files: []sandbox.FileEntry{
				{Name: "include/defs.h", Content: "#define X 42"},
			},
			expectError: true,
		},
		{
			name:     "Custom entrypoint in subdirectory that exists in files",
			language: "python",
			files: []sandbox.FileEntry{
				{Name: "pkg/util.py", Content: "VAL = 10"},
				{Name: "src/app.py", Content: "print(10)"},
			},
			entry:       "src/app.py",
			expectError: false,
		},
		{
			name:     "Custom entrypoint that does NOT exist in files",
			language: "python",
			files: []sandbox.FileEntry{
				{Name: "src/app.py", Content: "print(10)"},
			},
			entry:       "src/missing.py",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := sandbox.ValidateFileEntries(tc.language, tc.files, tc.entry)
			if tc.expectError && err == nil {
				t.Errorf("[%s] esperado erro, obtido nil", tc.name)
			}
			if !tc.expectError && err != nil {
				t.Errorf("[%s] nao esperava erro, obtido: %v", tc.name, err)
			}
		})
	}
}
