package handlers

import (
	"net/http"
	"os"
	"sige/internal/api/presenter"
	"sige/internal/cgroups"
	"sige/internal/engine"
	"time"
)

// HandleHealth processa requisições de liveness probe (GET /health).
func HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método HTTP não permitido", nil)
		return
	}
	presenter.RenderJSON(w, http.StatusOK, map[string]any{
		"status":    "UP",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleReady processa requisições de readiness probe (GET /ready).
// Verifica se o kernel possui cgroups v2, se o binário sige-launch existe e se o pool tem vagas.
func HandleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Método HTTP não permitido", nil)
		return
	}

	checks := make(map[string]string)
	allOk := true

	// 1. Verifica suporte a cgroups v2
	if cgroups.VerifyCgroupsVersion() {
		checks["cgroups_v2"] = "ok"
	} else {
		checks["cgroups_v2"] = "failed: host não está em Unified cgroup v2 mode"
		allOk = false
	}

	// 2. Verifica binário do sandbox
	launcherPath := os.Getenv("SIGE_EXECUTABLE")
	if launcherPath == "" {
		if exe, err := os.Executable(); err == nil {
			launcherPath = exe
		} else {
			launcherPath = "/opt/sige/sige-launch"
		}
	}
	if info, err := os.Stat(launcherPath); err == nil && !info.IsDir() {
		checks["launcher_binary"] = "ok"
	} else {
		checks["launcher_binary"] = "warning: binário do launcher não localizado em " + launcherPath
	}

	// 3. Verifica capacidade do pool
	active, total := engine.GlobalPool().Stats()
	if active >= int64(total) {
		checks["pool_capacity"] = "saturated"
		allOk = false
	} else {
		checks["pool_capacity"] = "ok"
	}

	status := http.StatusOK
	if !allOk {
		status = http.StatusServiceUnavailable
	}

	presenter.RenderJSON(w, status, map[string]any{
		"ready":     allOk,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"checks":    checks,
	})
}
