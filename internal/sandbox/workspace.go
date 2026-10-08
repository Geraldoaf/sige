package sandbox

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"sige/internal/constants"
)

var safeFilenameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
var safePathComponentRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// FileEntry representa um arquivo individual do projeto.
type FileEntry struct {
	Name          string `json:"name"`
	Content       string `json:"content,omitempty"`
	ContentBase64 string `json:"content_base64,omitempty"`
}

// IsSafeFilename valida se o nome do arquivo contém apenas caracteres seguros (arquivo único, sem barras).
func IsSafeFilename(name string) bool {
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, "-") {
		return false
	}
	return safeFilenameRe.MatchString(name)
}

const maxRelativePathDepth = 10

// IsSafeRelativePath valida caminhos relativos seguros sem permitir path traversal (../ ou barras iniciais).
func IsSafeRelativePath(relPath string) bool {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" || strings.HasPrefix(relPath, "/") || strings.HasPrefix(relPath, "\\") ||
		strings.HasSuffix(relPath, "/") || strings.HasSuffix(relPath, "\\") {
		return false
	}
	normalized := filepath.ToSlash(relPath)
	if strings.Contains(normalized, "..") || strings.Contains(normalized, "\x00") {
		return false
	}
	padded := "/" + normalized + "/"
	if strings.Contains(padded, "/./") || strings.Contains(padded, "//") {
		return false
	}
	clean := filepath.Clean(normalized)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	parts := strings.Split(clean, "/")
	if len(parts) > maxRelativePathDepth {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, "-") ||
			len(part) > 255 || !safePathComponentRe.MatchString(part) {
			return false
		}
	}
	return true
}

// ValidateFileEntries verifica duplicatas, colisões arquivo/diretório e nomes reservados na lista de arquivos.
func ValidateFileEntries(language string, files []FileEntry, entryFilename string) error {
	seenFiles := make(map[string]bool, len(files))
	seenDirs := make(map[string]bool)
	isCompiled := language == "c" || language == "cpp" || language == "c++"
	hasCompilableSource := false

	for i, f := range files {
		normName := filepath.ToSlash(strings.TrimSpace(f.Name))
		if !IsSafeRelativePath(normName) {
			return fmt.Errorf("Field 'files[%d].name' is invalid: only safe relative paths without '..' or leading '-' are allowed", i)
		}
		if isCompiled && (normName == "solution" || strings.HasPrefix(normName, "solution/")) {
			return fmt.Errorf("Field 'files[%d].name' cannot use reserved binary name 'solution'", i)
		}
		if seenFiles[normName] {
			return fmt.Errorf("Duplicate file path in 'files': '%s'", normName)
		}
		if seenDirs[normName] {
			return fmt.Errorf("File path '%s' in 'files' collides with a directory of the same name", normName)
		}

		dir := filepath.ToSlash(filepath.Dir(normName))
		for dir != "." && dir != "/" && dir != "" {
			if seenFiles[dir] {
				return fmt.Errorf("Directory '%s' in '%s' collides with an existing file of the same name", dir, normName)
			}
			seenDirs[dir] = true
			dir = filepath.ToSlash(filepath.Dir(dir))
		}

		seenFiles[normName] = true

		ext := strings.ToLower(filepath.Ext(normName))
		if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
			hasCompilableSource = true
		}
	}

	if isCompiled && len(files) > 0 && !hasCompilableSource {
		return errors.New("Compiled languages (c, cpp) require at least one source file (.c, .cpp, .cc, .cxx) in 'files'")
	}

	if entryFilename != "" && len(files) > 0 {
		normEntry := filepath.ToSlash(strings.TrimSpace(entryFilename))
		if !IsSafeRelativePath(normEntry) {
			return errors.New("Field 'filename' is invalid: only safe relative paths without '..' are allowed")
		}
		if !isCompiled && !seenFiles[normEntry] {
			return fmt.Errorf("Entrypoint 'filename' ('%s') was not found in 'files'", normEntry)
		}
	}

	return nil
}

// CompilationError representa um erro de compilação em código C/C++.
type CompilationError struct {
	Stderr string
}

func (e *CompilationError) Error() string {
	return "compilation error: " + e.Stderr
}

// CompileSources compila múltiplos arquivos C/C++ de forma isolada dentro do sandbox.
func CompileSources(language, hostWd string, sandboxSourcePaths []string, sandboxBinaryPath string) error {
	return CompileSourcesWithFlags(language, hostWd, sandboxSourcePaths, sandboxBinaryPath, nil)
}

// CompileSourcesWithFlags compila múltiplos arquivos C/C++ incluindo flags customizadas de forma isolada dentro do sandbox.
func CompileSourcesWithFlags(language, hostWd string, sandboxSourcePaths []string, sandboxBinaryPath string, compileFlags []string) error {
	if err := ValidateCompileFlags(language, compileFlags); err != nil {
		return err
	}

	var cleanFlags []string
	for _, f := range compileFlags {
		if trimmed := strings.TrimSpace(f); trimmed != "" {
			cleanFlags = append(cleanFlags, trimmed)
		}
	}

	var compiler string
	var compileArgs []string

	switch language {
	case "c":
		compiler = "/usr/bin/gcc"
		compileArgs = []string{"-I/workspace"}
		compileArgs = append(compileArgs, sandboxSourcePaths...)
		compileArgs = append(compileArgs, cleanFlags...)
		compileArgs = append(compileArgs, "-o", sandboxBinaryPath)
	case "cpp", "c++":
		compiler = "/usr/bin/g++"
		compileArgs = []string{"-I/workspace"}
		compileArgs = append(compileArgs, sandboxSourcePaths...)
		compileArgs = append(compileArgs, cleanFlags...)
		compileArgs = append(compileArgs, "-o", sandboxBinaryPath)
	default:
		return fmt.Errorf("unsupported compilation language: %s", language)
	}

	cfg := Config{
		Name:              fmt.Sprintf("compile-%d-%d", time.Now().UnixNano(), os.Getpid()),
		Workspace:         hostWd,
		WorkspaceWritable: true,
		MemoryMB:          constants.DefaultCompileMemoryMB,
		CPU:               fmt.Sprintf("%d%%", constants.DefaultCompileCPUPercent),
		TimeoutSec:        constants.DefaultCompileTimeoutSec,
		TmpLimitMB:        constants.DefaultCompileTmpLimitMB,
		MaxFileSizeMB:     constants.DefaultCompileMaxFileMB,
		MaxOpenFiles:      constants.DefaultCompileMaxOpenFiles,
	}

	result, err := Execute(cfg, compiler, compileArgs)
	if err != nil {
		if errors.Is(err, ErrAtCapacity) {
			return err
		}
		if result.Status == "timeout" {
			return &CompilationError{Stderr: fmt.Sprintf("Compilation timed out after %d seconds", constants.DefaultCompileTimeoutSec)}
		}
		// Se result.Status for vazio e err não for erro de exit code, é erro de infraestrutura
		if result.Status == "" && !strings.Contains(err.Error(), "exit status") {
			return err
		}
		output := strings.TrimSpace(result.Stdout + result.Stderr)
		if output == "" {
			output = err.Error()
		}
		return &CompilationError{Stderr: output}
	}

	return nil
}

// CompileSource compila código C/C++ de forma isolada dentro do sandbox (mantido para retrocompatibilidade).
func CompileSource(language, hostWd, sandboxSourcePath, sandboxBinaryPath string) error {
	return CompileSources(language, hostWd, []string{sandboxSourcePath}, sandboxBinaryPath)
}

// ResolveDefaultFilename retorna o nome do arquivo adequado para cada linguagem.
func ResolveDefaultFilename(language, userFilename string) string {
	userFilename = strings.TrimSpace(userFilename)
	if userFilename != "" && IsSafeFilename(userFilename) {
		return userFilename
	}

	switch language {
	case "python", "python3":
		return "main.py"
	case "bash", "sh":
		return "script.sh"
	case "c":
		return "solution.c"
	case "cpp", "c++":
		return "solution.cpp"
	default:
		return "script.sh"
	}
}

// WorkspaceSpec especifica as propriedades para materialização de um workspace.
type WorkspaceSpec struct {
	Language     string
	Files        []FileEntry
	Filename     string
	Code         string
	FileBase64   string
	CompileFlags []string
	ExecID       string
	BaseDir      string
}

// PreparedWorkspace contém os metadados do workspace pronto para execução no sandbox.
type PreparedWorkspace struct {
	Command string
	Args    []string
	HostWd  string
	Cleanup func()
}

// PrepareWorkspace prepara o diretório efêmero, decodifica código e executa compilação prévia se necessário.
func PrepareWorkspace(spec WorkspaceSpec) (*PreparedWorkspace, error) {
	lang := strings.ToLower(strings.TrimSpace(spec.Language))

	// Valida lista de arquivos caso fornecida explicitamente
	if len(spec.Files) > 0 {
		if err := ValidateFileEntries(lang, spec.Files, spec.Filename); err != nil {
			return nil, err
		}
	}

	if err := ValidateCompileFlags(lang, spec.CompileFlags); err != nil {
		return nil, err
	}

	// Normalização de arquivos caso spec.Files não tenha sido fornecido diretamente
	files := spec.Files
	if len(files) == 0 {
		resolvedFilename := ResolveDefaultFilename(lang, spec.Filename)
		files = []FileEntry{{
			Name:          resolvedFilename,
			Content:       spec.Code,
			ContentBase64: spec.FileBase64,
		}}
	}

	baseDir := strings.TrimSpace(spec.BaseDir)
	if baseDir == "" {
		baseDir = os.Getenv("SIGE_WORKSPACE_DIR")
	}
	if baseDir == "" {
		if info, err := os.Stat("/workspace"); err == nil && info.IsDir() {
			baseDir = "/workspace"
		} else {
			baseDir = filepath.Join(os.TempDir(), "sige-workspace")
		}
	}

	var hostWd string
	var err error
	if spec.ExecID != "" {
		hostWd = filepath.Join(baseDir, spec.ExecID)
		if err := os.MkdirAll(hostWd, 0777); err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("error creating workspace directory: %w", err)
		}
	} else {
		if err := os.MkdirAll(baseDir, 0755); err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("error creating base workspace directory: %w", err)
		}
		hostWd, err = os.MkdirTemp(baseDir, "exec-")
		if err != nil {
			return nil, fmt.Errorf("error creating temporary workspace directory: %w", err)
		}
	}

	// Permissão 0777 durante preparação/compilação para que o compilador (nobody)
	// possa listar diretórios e gravar /workspace/solution; ao final é reduzida para 0755.
	if err := os.Chmod(hostWd, 0777); err != nil {
		_ = os.RemoveAll(hostWd)
		return nil, fmt.Errorf("error setting workspace directory permissions: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(hostWd)
	}

	var sourceFiles []string
	entryFile := filepath.ToSlash(strings.TrimSpace(files[0].Name))
	if spec.Filename != "" && IsSafeRelativePath(spec.Filename) {
		entryFile = filepath.ToSlash(strings.TrimSpace(spec.Filename))
	}

	for _, f := range files {
		normName := filepath.ToSlash(strings.TrimSpace(f.Name))
		if !IsSafeRelativePath(normName) {
			cleanup()
			return nil, fmt.Errorf("invalid or unsafe filename in 'files': %s", f.Name)
		}

		var fileData []byte
		if f.Content != "" {
			fileData = []byte(f.Content)
		} else if f.ContentBase64 != "" {
			decoded, err := base64.StdEncoding.DecodeString(f.ContentBase64)
			if err != nil {
				cleanup()
				return nil, fmt.Errorf("invalid base64 in file '%s': %w", f.Name, err)
			}
			fileData = decoded
		} else {
			fileData = []byte{}
		}

		targetFilePath := filepath.Join(hostWd, filepath.FromSlash(normName))
		targetDir := filepath.Dir(targetFilePath)
		if targetDir != hostWd {
			if err := mkdirAllWithMode(hostWd, targetDir, 0755); err != nil {
				cleanup()
				return nil, fmt.Errorf("error creating directory for file '%s': %w", f.Name, err)
			}
		}

		if err := os.WriteFile(targetFilePath, fileData, 0644); err != nil {
			cleanup()
			return nil, fmt.Errorf("error writing file '%s': %w", f.Name, err)
		}

		ext := strings.ToLower(filepath.Ext(normName))
		if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".cxx" {
			sourceFiles = append(sourceFiles, filepath.Join("/workspace", normName))
		}
	}

	sandboxFilePath := filepath.Join("/workspace", entryFile)
	var command string
	var args []string

	switch lang {
	case "python", "python3":
		command = "/usr/bin/python3"
		args = []string{sandboxFilePath}
	case "bash", "sh":
		command = "/bin/bash"
		args = []string{sandboxFilePath}
	case "c", "cpp", "c++":
		if len(sourceFiles) == 0 {
			sourceFiles = []string{sandboxFilePath}
		}
		sandboxBinaryPath := filepath.Join("/workspace", "solution")
		if err := CompileSourcesWithFlags(lang, hostWd, sourceFiles, sandboxBinaryPath, spec.CompileFlags); err != nil {
			cleanup()
			return nil, err
		}
		command = sandboxBinaryPath
		args = []string{}
	default:
		cleanup()
		return nil, fmt.Errorf("language '%s' not supported", spec.Language)
	}

	// Após compilação (ou materialização de scripts), define 0755 em hostWd:
	// permite leitura/listagem (r-x) por nobody (necessário para importlib do Python)
	// e remove permissão de escrita mesmo no nível de inode (além do mount MS_RDONLY).
	_ = os.Chmod(hostWd, 0755)

	return &PreparedWorkspace{
		Command: command,
		Args:    args,
		HostWd:  hostWd,
		Cleanup: cleanup,
	}, nil
}

// mkdirAllWithMode cria subdiretórios dentro de rootDir garantindo permissão explícita (sem interferência de umask).
func mkdirAllWithMode(rootDir, targetDir string, perm os.FileMode) error {
	if err := os.MkdirAll(targetDir, perm); err != nil {
		return err
	}
	rel, err := filepath.Rel(rootDir, targetDir)
	if err != nil {
		return err
	}
	curr := rootDir
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." {
			continue
		}
		curr = filepath.Join(curr, part)
		if err := os.Chmod(curr, perm); err != nil {
			return err
		}
	}
	return nil
}
