package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"sige/internal/api"
	"sige/internal/config"
	"strings"
	"testing"
)

func setupConfig(t *testing.T, mode string) func() {
	cfg := config.DefaultConfig{
		MemoryMB:      50,
		CPU:           "",
		TimeoutSec:    3,
		TmpLimitMB:    64,
		MaxFileSizeMB: 15,
		MaxOpenFiles:  256,
		APIMode:       mode,
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		t.Fatalf("Erro ao criar config.json para teste: %v", err)
	}
	return func() {
		_ = os.Remove("config.json")
	}
}

func TestRealCaseAPI_Interpreter_QuickSort(t *testing.T) {
	cleanup := setupConfig(t, "interpreter")
	defer cleanup()

	code, err := os.ReadFile("real_cases/01_quicksort.py")
	if err != nil {
		t.Fatalf("Erro ao ler script do caso real: %v", err)
	}

	reqBody, _ := json.Marshal(api.ExecuteRequest{
		Language: "python",
		Code:     string(code),
	})

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	server := http.HandlerFunc(api.HandleExecute)
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Esperado status 200 OK na API, obtido %d: %s", rr.Code, rr.Body.String())
	}

	var resp api.ExecuteResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Erro ao deserializar JSON da resposta da API: %v", err)
	}

	if resp.Execution != nil {
		t.Logf("[API Result] QuickSort Mode: %s, Result: %s, ExitCode: %d", resp.Mode, resp.Result, resp.Execution.ExitCode)
	} else {
		t.Logf("[API Result] QuickSort Mode: %s, Result: %s", resp.Mode, resp.Result)
	}
}

func TestRealCaseAPI_SingleEvaluation_SuccessMatch(t *testing.T) {
	cleanup := setupConfig(t, "single_evaluation")
	defer cleanup()

	pythonCode := `import sys
lines = sys.stdin.read().split()
if lines:
    a, b = int(lines[0]), int(lines[1])
    print(a + b)
`

	reqBody, _ := json.Marshal(api.ExecuteRequest{
		Language:       "python",
		Code:           pythonCode,
		Stdin:          "15 27\n",
		ExpectedStdout: "42\n",
	})

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	server := http.HandlerFunc(api.HandleExecute)
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Esperado status 200 OK na API, obtido %d: %s", rr.Code, rr.Body.String())
	}

	var resp api.ExecuteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)

	t.Logf("[API SingleEval] Mode: %s, Result: %s, Passed: %d/%d", resp.Mode, resp.Result, resp.PassedCount, resp.TotalCount)
}

func TestRealCaseAPI_MultiEvaluation_Suite(t *testing.T) {
	cleanup := setupConfig(t, "multi_evaluation")
	defer cleanup()

	pythonCode := `import sys
num = int(sys.stdin.read().strip())
if num % 2 == 0:
    print("EVEN")
else:
    print("ODD")
`

	reqBody, _ := json.Marshal(api.ExecuteRequest{
		Language: "python",
		Code:     pythonCode,
		TestCases: []api.TestCase{
			{Stdin: "4\n", ExpectedStdout: "EVEN\n"},
			{Stdin: "7\n", ExpectedStdout: "ODD\n"},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	rr := httptest.NewRecorder()

	server := http.HandlerFunc(api.HandleExecute)
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Esperado status 200 OK na API no modo multi_evaluation, obtido %d: %s", rr.Code, rr.Body.String())
	}

	var resp api.ExecuteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	t.Logf("[API MultiEval] Mode: %s, Result: %s, Passed: %d/%d", resp.Mode, resp.Result, resp.PassedCount, resp.TotalCount)
}

func TestRealCaseAPI_ValidationErrors(t *testing.T) {
	cleanup := setupConfig(t, "interpreter")
	defer cleanup()

	reqGet := httptest.NewRequest(http.MethodGet, "/execute", nil)
	rrGet := httptest.NewRecorder()
	api.HandleExecute(rrGet, reqGet)
	if rrGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("Esperado 405 Method Not Allowed, obtido %d", rrGet.Code)
	}

	reqBadJSON := httptest.NewRequest(http.MethodPost, "/execute", strings.NewReader("{bad json"))
	rrBadJSON := httptest.NewRecorder()
	api.HandleExecute(rrBadJSON, reqBadJSON)
	if rrBadJSON.Code != http.StatusBadRequest {
		t.Errorf("Esperado 400 Bad Request para JSON malformado, obtido %d", rrBadJSON.Code)
	}
}

func TestRealCaseCLI_BinaryCommands(t *testing.T) {

	if _, err := os.Stat("../sige"); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", "../sige", "../main.go")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Erro ao compilar binário sige para testes CLI: %v", err)
		}
	}

	outHelp, err := exec.Command("../sige", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("Comando sige --help falhou: %v", err)
	}
	if !strings.Contains(string(outHelp), "Available Commands:") {
		t.Errorf("Saída do --help não contém a seção de comandos: %s", string(outHelp))
	}

	outCg, err := exec.Command("../sige", "cgroups-version").CombinedOutput()
	if err != nil {
		t.Fatalf("Comando sige cgroups-version falhou: %v", err)
	}
	if !strings.Contains(string(outCg), "cgroup version") {
		t.Errorf("Saída do cgroups-version inesperada: %s", string(outCg))
	}

	outRunTaskHelp, err := exec.Command("../sige", "run-task", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("Comando sige run-task --help falhou: %v", err)
	}
	if !strings.Contains(string(outRunTaskHelp), "cgroup v2 sandbox") {
		t.Errorf("Saída do run-task --help inesperada: %s", string(outRunTaskHelp))
	}
}
