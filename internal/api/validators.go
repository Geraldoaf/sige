package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"sige/internal/config"
	"sige/internal/sandbox"
	"strconv"
	"strings"
)

// Limites máximos por campo da requisição
const (
	maxFilesCount          = 50
	maxTotalFilesBytes     = 10 << 20 // 10MB total de arquivos
	maxTestCases           = 20
	maxCodeBytes           = 1 << 20 // 1MB de código-fonte
	maxFileBase64Bytes     = 1 << 21 // 2MB em base64
	maxStdinBytes          = 1 << 20 // 1MB de stdin
	maxExpectedStdoutBytes = 1 << 20 // 1MB de saída esperada
	maxFilenameBytes       = 255     // Limite do sistema de arquivos
)

// cpuQuotaRe valida formatos de quota de CPU ("10", "10%", ou "10000 100000").
var cpuQuotaRe = regexp.MustCompile(`^(\d+%?|\d+\s+\d+)$`)

// validateSizes verifica se os tamanhos dos campos da requisição respeitam os limites permitidos.
func validateSizes(req *ExecuteRequest) error {
	if len(req.Files) > maxFilesCount {
		return fmt.Errorf("Too many files in 'files': maximum allowed is %d.", maxFilesCount)
	}
	var totalFilesBytes int
	for i, f := range req.Files {
		totalFilesBytes += len(f.Content) + len(f.ContentBase64)
		if len(f.Name) > maxFilenameBytes {
			return fmt.Errorf("Filename of files[%d] is too long: maximum allowed is %d bytes.", i, maxFilenameBytes)
		}
	}
	if totalFilesBytes > maxTotalFilesBytes {
		return fmt.Errorf("Total size of 'files' is too large: maximum allowed is %d bytes.", maxTotalFilesBytes)
	}

	if len(req.Code) > maxCodeBytes {
		return fmt.Errorf("Field 'code' is too large: maximum allowed is %d bytes.", maxCodeBytes)
	}
	if len(req.FileBase64) > maxFileBase64Bytes {
		return fmt.Errorf("Field 'file_base64' is too large: maximum allowed is %d bytes.", maxFileBase64Bytes)
	}
	if len(req.Filename) > maxFilenameBytes {
		return fmt.Errorf("Field 'filename' is too long: maximum allowed is %d bytes.", maxFilenameBytes)
	}
	if len(req.Stdin) > maxStdinBytes {
		return fmt.Errorf("Field 'stdin' is too large: maximum allowed is %d bytes.", maxStdinBytes)
	}
	if len(req.ExpectedStdout) > maxExpectedStdoutBytes {
		return fmt.Errorf("Field 'expected_stdout' is too large: maximum allowed is %d bytes.", maxExpectedStdoutBytes)
	}
	for i, tc := range req.TestCases {
		if len(tc.Stdin) > maxStdinBytes {
			return fmt.Errorf("Field 'stdin' of test_cases[%d] is too large: maximum allowed is %d bytes.", i, maxStdinBytes)
		}
		if len(tc.ExpectedStdout) > maxExpectedStdoutBytes {
			return fmt.Errorf("Field 'expected_stdout' of test_cases[%d] is too large: maximum allowed is %d bytes.", i, maxExpectedStdoutBytes)
		}
	}
	return nil
}

// validateRequest valida a consistência e parâmetros do payload JSON recebido.
func validateRequest(req *ExecuteRequest, apiMode string) error {
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if req.Language == "" {
		return errors.New("Field 'language' is required")
	}

	switch req.Language {
	case "python", "python3", "bash", "sh", "c", "cpp", "c++":
	default:
		return fmt.Errorf("Field 'language' is unsupported: '%s'. Supported languages: python, bash, c, cpp", req.Language)
	}

	if len(req.Files) == 0 && req.Code == "" && req.FileBase64 == "" {
		return errors.New("You must provide code in 'code', 'file_base64', or a list of files in 'files'")
	}

	if err := validateSizes(req); err != nil {
		return err
	}

	if len(req.Files) == 0 {
		if req.Filename != "" && !sandbox.IsSafeFilename(req.Filename) {
			return errors.New("Field 'filename' is invalid: only letters, digits, '.', '-' and '_' are allowed")
		}
	} else {
		for i, f := range req.Files {
			if !sandbox.IsSafeRelativePath(f.Name) {
				return fmt.Errorf("Field 'files[%d].name' is invalid: only safe relative paths without '..' are allowed", i)
			}
			if f.Content == "" && f.ContentBase64 == "" {
				return fmt.Errorf("Field 'files[%d]' must contain 'content' or 'content_base64'", i)
			}
			if f.ContentBase64 != "" {
				if _, err := base64.StdEncoding.DecodeString(f.ContentBase64); err != nil {
					return fmt.Errorf("Field 'files[%d].content_base64' is not valid base64: %v", i, err)
				}
			}
		}
		if err := sandbox.ValidateFileEntries(req.Language, req.Files, req.Filename); err != nil {
			return err
		}
	}

	if req.FileBase64 != "" {
		if _, err := base64.StdEncoding.DecodeString(req.FileBase64); err != nil {
			return fmt.Errorf("Field 'file_base64' is not valid base64: %v", err)
		}
	}

	if err := sandbox.ValidateCompileFlags(req.Language, req.GetCompileFlags()); err != nil {
		return err
	}

	if req.CPU != "" {
		trimmedCPU := strings.TrimSpace(req.CPU)
		if !cpuQuotaRe.MatchString(trimmedCPU) {
			return errors.New("Field 'cpu' must be a percentage (e.g. '10' or '10%') or cgroup format (e.g. '10000 100000')")
		}
		parts := strings.Fields(trimmedCPU)
		if len(parts) == 2 {
			quota, _ := strconv.Atoi(parts[0])
			period, _ := strconv.Atoi(parts[1])
			if quota < 1000 || period < 1000 || period > 1000000 {
				return errors.New("Field 'cpu' format 'quota period' requires quota >= 1000 and period between 1000 and 1000000 microseconds")
			}
		} else if strings.HasSuffix(trimmedCPU, "%") {
			pct, _ := strconv.Atoi(strings.TrimSuffix(trimmedCPU, "%"))
			if pct <= 0 || pct > 10000 {
				return errors.New("Field 'cpu' percentage must be positive and reasonable (e.g. 1% to 1000%)")
			}
		} else {
			pct, err := strconv.Atoi(trimmedCPU)
			if err == nil && (pct <= 0 || pct > 10000) {
				return errors.New("Field 'cpu' value must be positive (e.g. 1 to 1000)")
			}
		}
	}

	switch apiMode {
	case "interpreter":
		if len(req.TestCases) > 0 || req.ExpectedStdout != "" {
			return errors.New("This API is configured in 'interpreter' mode and does not accept test evaluation parameters (expected_stdout or test_cases).")
		}
		if req.MemoryMB > 0 || req.CPU != "" || req.TimeoutSec > 0 || req.TmpLimitMB > 0 || req.MaxFileSizeMB > 0 || req.MaxOpenFiles > 0 {
			return errors.New("This API is configured in 'interpreter' mode and does not allow resource limits overrides.")
		}
	case "single_evaluation":
		if len(req.TestCases) > 0 {
			return errors.New("This API is configured in 'single_evaluation' mode and does not accept multiple test_cases.")
		}
		if req.ExpectedStdout == "" {
			return errors.New("This API is configured in 'single_evaluation' mode and requires field 'expected_stdout'.")
		}
	case "multi_evaluation":
		if len(req.TestCases) == 0 {
			return errors.New("This API is configured in 'multi_evaluation' mode and requires 'test_cases'.")
		}
		if len(req.TestCases) > maxTestCases {
			return fmt.Errorf("Too many test_cases: maximum allowed is %d.", maxTestCases)
		}
	default:
		return fmt.Errorf("API mode '%s' is unknown.", apiMode)
	}

	return nil
}

// resolveLimits resolve e aplica os tetos de recursos para a execução atual.
func resolveLimits(req *ExecuteRequest, defaultCfg *config.DefaultConfig, apiMode string) (
	mem int64, cpu string, timeout, tmpLimit, fileSize, openFiles int,
) {
	base := *defaultCfg
	if apiMode == "single_evaluation" || apiMode == "multi_evaluation" {
		custom := config.CustomLimits{
			MemoryMB:      req.MemoryMB,
			CPU:           req.CPU,
			TimeoutSec:    req.TimeoutSec,
			TmpLimitMB:    req.TmpLimitMB,
			MaxFileSizeMB: req.MaxFileSizeMB,
			MaxOpenFiles:  req.MaxOpenFiles,
		}
		base = config.MergeLimits(base, custom)
	} else {
		base = config.ClampToCeilings(base)
	}
	return base.MemoryMB, base.CPU, base.TimeoutSec, base.TmpLimitMB, base.MaxFileSizeMB, base.MaxOpenFiles
}
