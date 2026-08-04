package api

import (
	"errors"
	"fmt"
	"sige/internal/config"
	"strings"
)

func validateRequest(req *ExecuteRequest, apiMode string) error {
	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if req.Language == "" {
		return errors.New("Field 'language' is required")
	}

	if req.Code == "" && req.FileBase64 == "" {
		return errors.New("You must provide code in 'code' or a base64 file in 'file_base64'")
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
	default:
		return fmt.Errorf("API mode '%s' is unknown.", apiMode)
	}

	return nil
}

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
	}
	return base.MemoryMB, base.CPU, base.TimeoutSec, base.TmpLimitMB, base.MaxFileSizeMB, base.MaxOpenFiles
}
