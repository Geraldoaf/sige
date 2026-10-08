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

func TestHandleValidate_Files(t *testing.T) {
	validPayload := map[string]any{
		"language": "c",
		"files": []map[string]any{
			{"name": "main.c", "content": "#include \"include/op.h\"\nint main(){return 0;}"},
			{"name": "include/op.h", "content": "int add(int a, int b);"},
			{"name": "src/op.c", "content": "int add(int a, int b){return a+b;}"},
		},
	}
	data, _ := json.Marshal(validPayload)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(data))
	rr := httptest.NewRecorder()
	handlers.HandleValidate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("esperado status 200 para payload multi-arquivo valido, obtido %d: %s", rr.Code, rr.Body.String())
	}

	invalidPayload := map[string]any{
		"language": "c",
		"files": []map[string]any{
			{"name": "solution", "content": "reserved binary name"},
			{"name": "main.c", "content": "int main(){return 0;}"},
		},
	}
	badData, _ := json.Marshal(invalidPayload)
	badReq := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(badData))
	badRR := httptest.NewRecorder()
	handlers.HandleValidate(badRR, badReq)
	if badRR.Code != http.StatusBadRequest {
		t.Fatalf("esperado status 400 para payload multi-arquivo com nome reservado, obtido %d", badRR.Code)
	}
}

func TestHandleValidate_CompileFlags(t *testing.T) {
	// 1. Array of flags for C
	payload1 := map[string]any{
		"language":      "c",
		"code":          "int main(){return 0;}",
		"compile_flags": []string{"-O3", "-std=c11", "-Wall"},
	}
	data1, _ := json.Marshal(payload1)
	req1 := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(data1))
	rr1 := httptest.NewRecorder()
	handlers.HandleValidate(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid compile_flags array, got %d: %s", rr1.Code, rr1.Body.String())
	}
	var resp1 map[string]any
	_ = json.Unmarshal(rr1.Body.Bytes(), &resp1)
	flags1, ok := resp1["compile_flags"].([]any)
	if !ok || len(flags1) != 3 {
		t.Errorf("expected compile_flags in response with 3 items, got %v", resp1["compile_flags"])
	}

	// 2. String of flags for C
	payload2 := map[string]any{
		"language":      "c",
		"code":          "int main(){return 0;}",
		"compile_flags": "-O2 -Wall -DDEBUG",
	}
	data2, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(data2))
	rr2 := httptest.NewRecorder()
	handlers.HandleValidate(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid compile_flags string, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// 3. Flags rejected for python
	payload3 := map[string]any{
		"language":      "python",
		"code":          "print(1)",
		"compile_flags": "-O3",
	}
	data3, _ := json.Marshal(payload3)
	req3 := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(data3))
	rr3 := httptest.NewRecorder()
	handlers.HandleValidate(rr3, req3)
	if rr3.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when compile_flags passed for python, got %d", rr3.Code)
	}

	// 4. Flags rejected for -o override
	payload4 := map[string]any{
		"language":      "c",
		"code":          "int main(){return 0;}",
		"compile_flags": []string{"-o", "evil"},
	}
	data4, _ := json.Marshal(payload4)
	req4 := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(data4))
	rr4 := httptest.NewRecorder()
	handlers.HandleValidate(rr4, req4)
	if rr4.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when compile_flags overrides output binary (-o), got %d", rr4.Code)
	}
}
