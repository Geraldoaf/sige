package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHomeHandler(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleHome)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("HomeHandler returned incorrect status: got %v, expected %v", status, http.StatusOK)
	}

	expected := "API SIGE - Running in API mode"
	if rr.Body.String() != expected {
		t.Errorf("HomeHandler returned incorrect body: got %q, expected %q", rr.Body.String(), expected)
	}
}

func setupConfigJSON() func() {
	configData := []byte(`{
		"memory_mb": 50,
		"cpu": "",
		"timeout_sec": 5,
		"tmp_limit_mb": 64,
		"max_file_size_mb": 15,
		"max_open_files": 256
	}`)
	_ = os.WriteFile("config.json", configData, 0644)
	return func() {
		_ = os.Remove("config.json")
	}
}

func TestExecuteHandlerInvalidMethod(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/execute", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("ExecuteHandler returned incorrect status for GET: got %v, expected %v", status, http.StatusMethodNotAllowed)
	}
}

func TestExecuteHandlerInvalidJSON(t *testing.T) {
	cleanup := setupConfigJSON()
	defer cleanup()

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBufferString("{invalid json"))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("ExecuteHandler returned incorrect status for invalid JSON: got %v, expected %v", status, http.StatusBadRequest)
	}
}

func TestExecuteHandlerMissingConfig(t *testing.T) {

	_ = os.Remove("config.json")

	reqBody, _ := json.Marshal(ExecuteRequest{
		Language: "python",
		Code:     "print('ok')",
	})

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusInternalServerError {
		t.Errorf("ExecuteHandler without config.json should fail with 500, got: %v", status)
	}
}

func setupConfigWithMode(mode string) func() {
	configData := []byte(fmt.Sprintf(`{
		"memory_mb": 50,
		"cpu": "",
		"timeout_sec": 5,
		"tmp_limit_mb": 64,
		"max_file_size_mb": 15,
		"max_open_files": 256,
		"api_mode": "%s"
	}`, mode))
	_ = os.WriteFile("config.json", configData, 0644)
	return func() {
		_ = os.Remove("config.json")
	}
}

func TestExecuteHandlerInterpreterModeBlockGrader(t *testing.T) {
	cleanup := setupConfigWithMode("interpreter")
	defer cleanup()

	reqBody, _ := json.Marshal(ExecuteRequest{
		Language:       "python",
		Code:           "print('ok')",
		ExpectedStdout: "ok",
	})

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("InterpreterModeBlockGrader: expected status 400 when sending expected_stdout, got: %v", status)
	}
}

func TestExecuteHandlerSingleEvaluationBlockInterpreter(t *testing.T) {
	cleanup := setupConfigWithMode("single_evaluation")
	defer cleanup()

	reqBody, _ := json.Marshal(ExecuteRequest{
		Language: "python",
		Code:     "print('ok')",
	})

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("SingleEvaluationBlockInterpreter: expected status 400 when omitting expected_stdout, got: %v", status)
	}
}

func TestExecuteHandlerMultiEvaluationBlockInterpreter(t *testing.T) {
	cleanup := setupConfigWithMode("multi_evaluation")
	defer cleanup()

	reqBody, _ := json.Marshal(ExecuteRequest{
		Language: "python",
		Code:     "print('ok')",
	})

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("MultiEvaluationBlockInterpreter: expected status 400 when omitting test_cases, got: %v", status)
	}
}

func TestExecuteHandlerInterpreterModeBlockOverrides(t *testing.T) {
	cleanup := setupConfigWithMode("interpreter")
	defer cleanup()

	reqBody, _ := json.Marshal(ExecuteRequest{
		Language: "python",
		Code:     "print('ok')",
		MemoryMB: 100,
	})

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("InterpreterModeBlockOverrides: expected status 400 when sending RAM limit override, got: %v", status)
	}
}

func TestExecuteHandlerEvaluationModeAllowOverrides(t *testing.T) {
	cleanup := setupConfigWithMode("single_evaluation")
	defer cleanup()

	reqBody, _ := json.Marshal(ExecuteRequest{
		Language:       "python",
		Code:           "print('ok')",
		ExpectedStdout: "ok",
		MemoryMB:       100,
	})

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status == http.StatusBadRequest {
		t.Errorf("EvaluationModeAllowOverrides: expected limits to be permitted (not returning 400), got: %v", status)
	}
}
