package handlers

import (
	"net/http"
	"os"
	"sige/internal/api/presenter"
	"sige/internal/engine"
)

// HandleCapacity retorna o status da fila de concorrência e ocupação dos sandboxes.
func HandleCapacity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método HTTP não permitido", nil)
		return
	}

	active, total := engine.GlobalPool().Stats()
	utilization := float64(0)
	if total > 0 {
		utilization = (float64(active) / float64(total)) * 100
	}

	mode := os.Getenv("SIGE_API_MODE")
	if mode == "" {
		mode = "interpreter"
	}

	presenter.RenderJSON(w, http.StatusOK, map[string]any{
		"active_sandboxes":         active,
		"max_concurrent_sandboxes": total,
		"available_slots":          total - int(active),
		"utilization_percent":      utilization,
		"api_mode":                 mode,
	})
}
