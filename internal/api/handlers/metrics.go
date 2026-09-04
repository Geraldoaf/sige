package handlers

import (
	"net/http"
	"sige/internal/api/presenter"
	"sige/internal/engine"
	"sync/atomic"
	"time"
)

var (
	startTime          = time.Now()
	totalExecutions    int64
	successExecutions  int64
	failedExecutions   int64
	timeoutExecutions  int64
	oomExecutions      int64
)

// RecordExecution contabiliza o desfecho de uma execução no sandbox.
func RecordExecution(status string) {
	atomic.AddInt64(&totalExecutions, 1)
	switch status {
	case "success":
		atomic.AddInt64(&successExecutions, 1)
	case "timeout":
		atomic.AddInt64(&failedExecutions, 1)
		atomic.AddInt64(&timeoutExecutions, 1)
	case "oom":
		atomic.AddInt64(&failedExecutions, 1)
		atomic.AddInt64(&oomExecutions, 1)
	default:
		atomic.AddInt64(&failedExecutions, 1)
	}
}

// HandleMetrics retorna dados estatísticos de saúde operacional e execuções.
func HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método HTTP não permitido", nil)
		return
	}

	active, total := engine.GlobalPool().Stats()

	presenter.RenderJSON(w, http.StatusOK, map[string]any{
		"uptime_seconds": int(time.Since(startTime).Seconds()),
		"executions": map[string]int64{
			"total":   atomic.LoadInt64(&totalExecutions),
			"success": atomic.LoadInt64(&successExecutions),
			"failed":  atomic.LoadInt64(&failedExecutions),
			"timeout": atomic.LoadInt64(&timeoutExecutions),
			"oom":     atomic.LoadInt64(&oomExecutions),
		},
		"sandboxes": map[string]any{
			"active": active,
			"limit":  total,
		},
	})
}
