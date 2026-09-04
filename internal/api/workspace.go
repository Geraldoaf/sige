package api

import (
	"errors"
	"sige/internal/sandbox"
)

// CompilationError é um alias para sandbox.CompilationError garantindo compatibilidade retroativa.
type CompilationError = sandbox.CompilationError

func isSafeFilename(name string) bool {
	return sandbox.IsSafeFilename(name)
}

func compileSource(language, hostWd, sandboxSourcePath, sandboxBinaryPath string) error {
	return sandbox.CompileSource(language, hostWd, sandboxSourcePath, sandboxBinaryPath)
}

// prepareWorkspace prepara o diretório efêmero delegando para sandbox.PrepareWorkspace.
func prepareWorkspace(req *ExecuteRequest, execID string) (command string, args []string, workspace string, cleanup func(), err error) {
	if req.Filename != "" && !isSafeFilename(req.Filename) {
		return "", nil, "", nil, errors.New("Invalid filename in 'filename'")
	}

	spec := sandbox.WorkspaceSpec{
		Language:   req.Language,
		Filename:   req.Filename,
		Code:       req.Code,
		FileBase64: req.FileBase64,
		ExecID:     execID,
	}

	res, err := sandbox.PrepareWorkspace(spec)
	if err != nil {
		return "", nil, "", nil, err
	}

	return res.Command, res.Args, res.HostWd, res.Cleanup, nil
}
