package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sige/internal/api/handlers"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handlers.HandleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtido %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["status"] != "UP" {
		t.Errorf("esperado status=UP, obtido %v", body["status"])
	}
}

func TestHandleLanguages(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/languages", nil)
	rr := httptest.NewRecorder()

	handlers.HandleLanguages(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtido %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	langs, ok := body["languages"].([]any)
	if !ok || len(langs) < 4 {
		t.Errorf("esperado pelo menos 4 linguagens, obtido %v", langs)
	}
}

func TestHandleCapacity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/capacity", nil)
	rr := httptest.NewRecorder()

	handlers.HandleCapacity(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtido %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["max_concurrent_sandboxes"] == nil {
		t.Errorf("campo max_concurrent_sandboxes ausente: %v", body)
	}
}

func TestHandleMetrics(t *testing.T) {
	handlers.RecordExecution("success")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handlers.HandleMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtido %d", rr.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	execs, ok := body["executions"].(map[string]any)
	if !ok || execs["total"] == float64(0) {
		t.Errorf("esperado contagem de execucoes maior que 0, obtido %v", execs)
	}
}

func TestHandleValidate_Valid(t *testing.T) {
	payload := map[string]any{
		"language": "python",
		"code":     "print('hello')",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(data))
	rr := httptest.NewRecorder()

	handlers.HandleValidate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("esperado status 200 para payload valido, obtido %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["valid"] != true {
		t.Errorf("esperado valid=true, obtido %v", body["valid"])
	}
}

func TestHandleValidate_InvalidLanguage(t *testing.T) {
	payload := map[string]any{
		"language": "invalid_lang_xyz",
		"code":     "print('hello')",
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(data))
	rr := httptest.NewRecorder()

	handlers.HandleValidate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("esperado status 400 para linguagem invalida, obtido %d", rr.Code)
	}
}
