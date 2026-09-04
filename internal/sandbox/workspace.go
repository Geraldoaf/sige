package sandbox

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"sige/internal/constants"
)

var safeFilenameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// IsSafeFilename valida se o nome do arquivo contém apenas caracteres seguros.
func IsSafeFilename(name string) bool {
	if name == "." || name == ".." {
		return false
	}
	return safeFilenameRe.MatchString(name)
}

// CompilationError representa um erro de compilação em código C/C++.
type CompilationError struct {
	Stderr string
}

func (e *CompilationError) Error() string {
	return "compilation error: " + e.Stderr
}

// CompileSource compila código C/C++ de forma isolada dentro do sandbox.
func CompileSource(language, hostWd, sandboxSourcePath, sandboxBinaryPath string) error {
	var compiler string
	var compileArgs []string

	switch language {
	case "c":
		compiler = "/usr/bin/gcc"
		compileArgs = []string{"-O2", "-Wall", sandboxSourcePath, "-o", sandboxBinaryPath, "-lm"}
	case "cpp", "c++":
		compiler = "/usr/bin/g++"
		compileArgs = []string{"-O2", "-Wall", sandboxSourcePath, "-o", sandboxBinaryPath, "-lm"}
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
		if result.Status == "timeout" {
			return &CompilationError{Stderr: fmt.Sprintf("Compilation timed out after %d seconds", constants.DefaultCompileTimeoutSec)}
		}
		output := strings.TrimSpace(result.Stdout + result.Stderr)
		if output == "" {
			output = err.Error()
		}
		return &CompilationError{Stderr: output}
	}

	return nil
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
	Language   string
	Filename   string
	Code       string
	FileBase64 string
	ExecID     string
	BaseDir    string
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
	resolvedFilename := ResolveDefaultFilename(spec.Language, spec.Filename)

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

	execID := spec.ExecID
	if execID == "" {
		execID = fmt.Sprintf("exec-%d-%d", time.Now().UnixNano(), os.Getpid())
	}

	hostWd := filepath.Join(baseDir, execID)
	if err := os.MkdirAll(hostWd, 0703); err != nil {
		return nil, fmt.Errorf("error creating workspace directory: %w", err)
	}
	if err := os.Chmod(hostWd, 0703); err != nil {
		return nil, fmt.Errorf("error setting workspace directory permissions: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(hostWd)
	}

	var codeData []byte
	if spec.Code != "" {
		codeData = []byte(spec.Code)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(spec.FileBase64)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("invalid base64 format in 'file_base64': %w", err)
		}
		codeData = decoded
	}

	targetFilePath := filepath.Join(hostWd, resolvedFilename)
	if err := os.WriteFile(targetFilePath, codeData, 0644); err != nil {
		cleanup()
		return nil, fmt.Errorf("error writing file: %w", err)
	}

	sandboxFilePath := filepath.Join("/workspace", resolvedFilename)
	var command string
	var args []string

	switch spec.Language {
	case "python", "python3":
		command = "/usr/bin/python3"
		args = []string{sandboxFilePath}
	case "bash", "sh":
		command = "/bin/bash"
		args = []string{sandboxFilePath}
	case "c", "cpp", "c++":
		sandboxBinaryPath := filepath.Join("/workspace", "solution")
		if err := CompileSource(spec.Language, hostWd, sandboxFilePath, sandboxBinaryPath); err != nil {
			return nil, err
		}
		command = sandboxBinaryPath
		args = []string{}
	default:
		cleanup()
		return nil, fmt.Errorf("language '%s' not supported", spec.Language)
	}

	return &PreparedWorkspace{
		Command: command,
		Args:    args,
		HostWd:  hostWd,
		Cleanup: cleanup,
	}, nil
}
