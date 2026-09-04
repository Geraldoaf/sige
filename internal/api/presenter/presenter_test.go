package presenter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sige/internal/api/presenter"
	"testing"
)

func TestRenderError(t *testing.T) {
	rr := httptest.NewRecorder()
	presenter.RenderError(rr, http.StatusBadRequest, "INVALID_INPUT", "Campo obrigatório ausente", map[string]any{"field": "language"})

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("esperado status 400, obtido %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("esperado Content-Type application/json, obtido %s", ct)
	}

	var env presenter.ErrorEnvelope
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("erro no unmarshal do JSON de erro: %v", err)
	}
	if env.Error.Code != "INVALID_INPUT" {
		t.Errorf("esperado code INVALID_INPUT, obtido %s", env.Error.Code)
	}
	if env.Error.Details["field"] != "language" {
		t.Errorf("esperado field=language nos detalhes, obtido %v", env.Error.Details)
	}
}

func TestRenderJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	presenter.RenderJSON(rr, http.StatusOK, map[string]string{"status": "ok"})

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtido %d", rr.Code)
	}
	var data map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &data); err != nil {
		t.Fatalf("erro no unmarshal: %v", err)
	}
	if data["status"] != "ok" {
		t.Errorf("esperado status=ok, obtido %s", data["status"])
	}
}
