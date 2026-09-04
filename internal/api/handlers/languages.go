package handlers

import (
	"net/http"
	"sige/internal/api/presenter"
	"sige/internal/config"
)

type LanguageSpec struct {
	ID               string         `json:"id"`
	Aliases          []string       `json:"aliases"`
	Type             string         `json:"type"` // "interpreted" ou "compiled"
	DefaultExtension string         `json:"default_extension"`
	DefaultLimits    map[string]any `json:"default_limits"`
}

// HandleLanguages retorna o catálogo de linguagens suportadas e seus limites padrão.
func HandleLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método HTTP não permitido", nil)
		return
	}

	cfg, _ := config.Resolve("config.json")

	defaultLimits := map[string]any{
		"memory_mb":   cfg.MemoryMB,
		"cpu":         cfg.CPU,
		"timeout_sec": cfg.TimeoutSec,
	}

	languages := []LanguageSpec{
		{
			ID:               "python",
			Aliases:          []string{"python3", "py"},
			Type:             "interpreted",
			DefaultExtension: ".py",
			DefaultLimits:    defaultLimits,
		},
		{
			ID:               "c",
			Aliases:          []string{"c"},
			Type:             "compiled",
			DefaultExtension: ".c",
			DefaultLimits:    defaultLimits,
		},
		{
			ID:               "cpp",
			Aliases:          []string{"c++", "cxx"},
			Type:             "compiled",
			DefaultExtension: ".cpp",
			DefaultLimits:    defaultLimits,
		},
		{
			ID:               "bash",
			Aliases:          []string{"sh", "shell"},
			Type:             "interpreted",
			DefaultExtension: ".sh",
			DefaultLimits:    defaultLimits,
		},
	}

	presenter.RenderJSON(w, http.StatusOK, map[string]any{
		"languages": languages,
	})
}
