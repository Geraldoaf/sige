package api

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
	"sige/internal/sandbox"
)

// safeFilenameRe define o padrão de caracteres seguros para nomes de arquivos.
var safeFilenameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// isSafeFilename valida se o nome do arquivo contém apenas caracteres seguros.
func isSafeFilename(name string) bool {
	if name == "." || name == ".." {
		return false
	}
	return safeFilenameRe.MatchString(name)
}

type CompilationError struct {
	Stderr string
}

func (e *CompilationError) Error() string {
	return "compilation error: " + e.Stderr
}

// compileSource compila código C/C++ de forma isolada dentro do sandbox.
func compileSource(language, hostWd, sandboxSourcePath, sandboxBinaryPath string) error {
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

	cfg := sandbox.Config{
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

	result, err := sandbox.Execute(cfg, compiler, compileArgs)
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

// prepareWorkspace cria o diretório temporário, grava o código-fonte e compila caso necessário.
func prepareWorkspace(req *ExecuteRequest, execID string) (command string, args []string, workspace string, cleanup func(), err error) {
	resolvedFilename := "solution.py"
	switch req.Language {
	case "bash", "sh":
		resolvedFilename = "solution.sh"
	case "c":
		resolvedFilename = "solution.c"
	case "cpp", "c++":
		resolvedFilename = "solution.cpp"
	}

	if req.Filename != "" {
		if !isSafeFilename(req.Filename) {
			return "", nil, "", nil, errors.New("Invalid filename in 'filename'")
		}
		resolvedFilename = req.Filename
	}

	baseDir := os.Getenv("SIGE_WORKSPACE_DIR")
	if baseDir == "" {
		if info, err := os.Stat("/workspace"); err == nil && info.IsDir() {
			baseDir = "/workspace"
		} else {
			baseDir = filepath.Join(os.TempDir(), "sige-workspace")
		}
	}

	hostWd := filepath.Join(baseDir, execID)
	if err := os.MkdirAll(hostWd, 0703); err != nil {
		return "", nil, "", nil, fmt.Errorf("error creating workspace directory: %w", err)
	}
	if err := os.Chmod(hostWd, 0703); err != nil {
		return "", nil, "", nil, fmt.Errorf("error setting workspace directory permissions: %w", err)
	}

	cleanup = func() {
		os.RemoveAll(hostWd)
	}

	var codeData []byte
	if req.Code != "" {
		codeData = []byte(req.Code)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(req.FileBase64)
		if err != nil {
			cleanup()
			return "", nil, "", nil, fmt.Errorf("invalid base64 format in 'file_base64': %w", err)
		}
		codeData = decoded
	}

	targetFilePath := filepath.Join(hostWd, resolvedFilename)
	if err := os.WriteFile(targetFilePath, codeData, 0644); err != nil {
		cleanup()
		return "", nil, "", nil, fmt.Errorf("error writing file: %w", err)
	}

	sandboxFilePath := filepath.Join("/workspace", resolvedFilename)

	switch req.Language {
	case "python", "python3":
		command = "/usr/bin/python3"
		args = []string{sandboxFilePath}
	case "bash", "sh":
		command = "/bin/bash"
		args = []string{sandboxFilePath}
	case "c", "cpp", "c++":
		sandboxBinaryPath := filepath.Join("/workspace", "solution")
		if err := compileSource(req.Language, hostWd, sandboxFilePath, sandboxBinaryPath); err != nil {
			return "", nil, "", cleanup, err
		}
		command = sandboxBinaryPath
		args = []string{}
	default:
		cleanup()
		return "", nil, "", nil, fmt.Errorf("language '%s' not supported", req.Language)
	}

	return command, args, hostWd, cleanup, nil
}
