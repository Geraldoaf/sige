package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"sige/internal/api/presenter"
	"sige/internal/config"
	"sige/internal/sandbox"
)

// ValidateRequestDTO representa a requisição a ser validada.
type ValidateRequestDTO struct {
	Language       string              `json:"language"`
	Code           string              `json:"code"`
	FileBase64     string              `json:"file_base64"`
	Filename       string              `json:"filename"`
	Stdin          string              `json:"stdin"`
	ExpectedStdout string              `json:"expected_stdout"`
	MemoryMB       int64               `json:"memory_mb"`
	CPU            string              `json:"cpu"`
	TimeoutSec     int                 `json:"timeout_sec"`
	TmpLimitMB     int                 `json:"tmp_limit_mb"`
	MaxFileSizeMB  int                 `json:"max_file_size_mb"`
	MaxOpenFiles   int                 `json:"max_open_files"`
	TestCases      []map[string]string `json:"test_cases"`
}

// HandleValidate executa um dry-run sintático e estrutural da requisição sem criar cgroups ou containers.
func HandleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método HTTP não permitido", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB

	var req ValidateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		presenter.RenderError(w, http.StatusBadRequest, "INVALID_JSON", "JSON malformado ou payload superior a 10MB: "+err.Error(), nil)
		return
	}

	if req.Language == "" {
		presenter.RenderError(w, http.StatusBadRequest, "MISSING_FIELD", "Campo 'language' é obrigatório", map[string]any{"field": "language"})
		return
	}

	switch req.Language {
	case "python", "python3", "bash", "sh", "c", "cpp", "c++":
	default:
		presenter.RenderError(w, http.StatusBadRequest, "UNSUPPORTED_LANGUAGE", "Linguagem não suportada: "+req.Language, map[string]any{
			"supported": []string{"python", "bash", "c", "cpp"},
		})
		return
	}

	if req.Code == "" && req.FileBase64 == "" {
		presenter.RenderError(w, http.StatusBadRequest, "MISSING_CODE", "É obrigatório fornecer 'code' ou 'file_base64'", nil)
		return
	}

	if req.Filename != "" && !sandbox.IsSafeFilename(req.Filename) {
		presenter.RenderError(w, http.StatusBadRequest, "INVALID_FILENAME", "Nome de arquivo inválido: apenas letras, dígitos, '.', '-' e '_' são permitidos", nil)
		return
	}

	apiMode := os.Getenv("SIGE_API_MODE")
	if apiMode == "" {
		apiMode = "interpreter"
	}

	baseCfg, _ := config.Resolve("config.json")
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

	presenter.RenderJSON(w, http.StatusOK, map[string]any{
		"valid":               true,
		"language":            req.Language,
		"normalized_filename": sandbox.ResolveDefaultFilename(req.Language, req.Filename),
		"code_bytes":          codeBytes,
		"api_mode":            apiMode,
		"resolved_limits": map[string]any{
			"memory_mb":   resolvedCfg.MemoryMB,
			"cpu":         resolvedCfg.CPU,
			"timeout_sec": resolvedCfg.TimeoutSec,
		},
	})
}
