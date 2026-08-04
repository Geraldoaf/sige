package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func prepareWorkspace(req *ExecuteRequest, execID string) (command string, args []string, cleanup func(), err error) {
	resolvedFilename := "solution.py"
	if req.Language == "bash" {
		resolvedFilename = "solution.sh"
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

	switch req.Language {
	case "python", "python3":
		command = "/usr/bin/python3"
		args = []string{targetFilePath}
	case "bash", "sh":
		command = "/bin/bash"
		args = []string{targetFilePath}
	default:
		cleanup()
		return "", nil, nil, fmt.Errorf("language '%s' not supported", req.Language)
	}

	return command, args, cleanup, nil
}
