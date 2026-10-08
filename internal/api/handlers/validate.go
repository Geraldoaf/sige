package handlers

import (
	"encoding/json"
	"net/http"
	"sige/internal/api/presenter"
	"sige/internal/config"
	"sige/internal/sandbox"
	"strings"
)

// ValidateRequestDTO representa a requisição a ser validada.
type ValidateRequestDTO struct {
	Language       string               `json:"language"`
	Files          []sandbox.FileEntry  `json:"files,omitempty"`
	Code           string               `json:"code"`
	FileBase64     string               `json:"file_base64"`
	Filename       string               `json:"filename"`
	CompileFlags   sandbox.CompileFlags `json:"compile_flags,omitempty"`
	CompilerFlags  sandbox.CompileFlags `json:"compiler_flags,omitempty"`
	Flags          sandbox.CompileFlags `json:"flags,omitempty"`
	Stdin          string               `json:"stdin"`
	ExpectedStdout string               `json:"expected_stdout"`
	MemoryMB       int64                `json:"memory_mb"`
	CPU            string               `json:"cpu"`
	TimeoutSec     int                  `json:"timeout_sec"`
	TmpLimitMB     int                  `json:"tmp_limit_mb"`
	MaxFileSizeMB  int                  `json:"max_file_size_mb"`
	MaxOpenFiles   int                  `json:"max_open_files"`
	TestCases      []map[string]string  `json:"test_cases"`
}

func (r *ValidateRequestDTO) GetCompileFlags() []string {
	if len(r.CompileFlags) > 0 {
		return r.CompileFlags
	}
	if len(r.CompilerFlags) > 0 {
		return r.CompilerFlags
	}
	return r.Flags
}

// HandleValidate executa um dry-run sintático e estrutural da requisição sem criar cgroups ou containers.
func HandleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "HTTP method not allowed", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB

	var req ValidateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		presenter.RenderError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed JSON or payload exceeding 10MB: "+err.Error(), nil)
		return
	}

	req.Language = strings.ToLower(strings.TrimSpace(req.Language))
	if req.Language == "" {
		presenter.RenderError(w, http.StatusBadRequest, "MISSING_FIELD", "Field 'language' is required", map[string]any{"field": "language"})
		return
	}

	switch req.Language {
	case "python", "python3", "bash", "sh", "c", "cpp", "c++":
	default:
		presenter.RenderError(w, http.StatusBadRequest, "UNSUPPORTED_LANGUAGE", "Unsupported language: "+req.Language, map[string]any{
			"supported": []string{"python", "bash", "c", "cpp"},
		})
		return
	}

	if len(req.Files) == 0 && req.Code == "" && req.FileBase64 == "" {
		presenter.RenderError(w, http.StatusBadRequest, "MISSING_CODE", "Field 'code', 'file_base64', or 'files' is required", nil)
		return
	}

	if len(req.Files) == 0 {
		if req.Filename != "" && !sandbox.IsSafeFilename(req.Filename) {
			presenter.RenderError(w, http.StatusBadRequest, "INVALID_FILENAME", "Field 'filename' is invalid: only letters, digits, '.', '-' and '_' are allowed", nil)
			return
		}
	} else {
		if err := sandbox.ValidateFileEntries(req.Language, req.Files, req.Filename); err != nil {
			presenter.RenderError(w, http.StatusBadRequest, "INVALID_FILES", err.Error(), nil)
			return
		}
	}

	if err := sandbox.ValidateCompileFlags(req.Language, req.GetCompileFlags()); err != nil {
		presenter.RenderError(w, http.StatusBadRequest, "INVALID_COMPILE_FLAGS", err.Error(), nil)
		return
	}

	baseCfg, _ := config.Resolve("config.json")
	apiMode := strings.ToLower(strings.TrimSpace(baseCfg.APIMode))
	if apiMode == "" {
		apiMode = "interpreter"
	}
	customLimits := config.CustomLimits{
		MemoryMB:      req.MemoryMB,
		CPU:           req.CPU,
		TimeoutSec:    req.TimeoutSec,
		TmpLimitMB:    req.TmpLimitMB,
		MaxFileSizeMB: req.MaxFileSizeMB,
		MaxOpenFiles:  req.MaxOpenFiles,
	}
	resolvedCfg := config.MergeLimits(baseCfg, customLimits)

	codeBytes := len(req.Code)
	if req.FileBase64 != "" {
		codeBytes = len(req.FileBase64)
	}
	for _, f := range req.Files {
		codeBytes += len(f.Content) + len(f.ContentBase64)
	}

	normalizedFilename := sandbox.ResolveDefaultFilename(req.Language, req.Filename)
	if len(req.Files) > 0 {
		normalizedFilename = req.Files[0].Name
		if req.Filename != "" {
			normalizedFilename = req.Filename
		}
	}

	respMap := map[string]any{
		"valid":               true,
		"language":            req.Language,
		"normalized_filename": normalizedFilename,
		"files_count":         len(req.Files),
		"code_bytes":          codeBytes,
		"api_mode":            apiMode,
		"resolved_limits": map[string]any{
			"memory_mb":   resolvedCfg.MemoryMB,
			"cpu":         resolvedCfg.CPU,
			"timeout_sec": resolvedCfg.TimeoutSec,
		},
	}
	if flags := req.GetCompileFlags(); len(flags) > 0 {
		respMap["compile_flags"] = flags
	}
	presenter.RenderJSON(w, http.StatusOK, respMap)
}
