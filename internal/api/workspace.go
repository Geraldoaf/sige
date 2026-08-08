package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type CompilationError struct {
	Stderr string
}

func (e *CompilationError) Error() string {
	return "compilation error: " + e.Stderr
}

func compileSource(language, sourcePath, binaryPath string) error {
	var compiler string
	var compileArgs []string

	switch language {
	case "c":
		compiler = "gcc"
		compileArgs = []string{"-O2", "-Wall", sourcePath, "-o", binaryPath, "-lm"}
	case "cpp", "c++":
		compiler = "g++"
		compileArgs = []string{"-O2", "-Wall", sourcePath, "-o", binaryPath, "-lm"}
	default:
		return fmt.Errorf("unsupported compilation language: %s", language)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, compiler, compileArgs...)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &CompilationError{Stderr: "Compilation timed out after 10 seconds"}
		}
		return &CompilationError{Stderr: string(outBytes)}
	}

	return nil
}

func prepareWorkspace(req *ExecuteRequest, execID string) (command string, args []string, cleanup func(), err error) {
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
		if strings.Contains(req.Filename, "..") || strings.Contains(req.Filename, "/") || strings.Contains(req.Filename, "\\") {
			return "", nil, nil, errors.New("Invalid filename in 'filename'")
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
	if err := os.MkdirAll(hostWd, 0755); err != nil {
		return "", nil, nil, fmt.Errorf("error creating workspace directory: %w", err)
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
			return "", nil, nil, fmt.Errorf("invalid base64 format in 'file_base64': %w", err)
		}
		codeData = decoded
	}

	targetFilePath := filepath.Join(hostWd, resolvedFilename)
	if err := os.WriteFile(targetFilePath, codeData, 0644); err != nil {
		cleanup()
		return "", nil, nil, fmt.Errorf("error writing file: %w", err)
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
		binaryPath := filepath.Join(hostWd, "solution")
		if err := compileSource(req.Language, targetFilePath, binaryPath); err != nil {
			return "", nil, cleanup, err
		}
		command = "/workspace/solution"
		args = []string{}
	default:
		cleanup()
		return "", nil, nil, fmt.Errorf("language '%s' not supported", req.Language)
	}

	return command, args, cleanup, nil
}

