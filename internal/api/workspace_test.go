package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrepareWorkspaceCSuccess(t *testing.T) {
	req := ExecuteRequest{
		Language: "c",
		Code:     "#include <stdio.h>\nint main() { printf(\"Hello C\\n\"); return 0; }",
	}

	cmd, args, cleanup, err := prepareWorkspace(&req, "test-exec-c")
	if err != nil {
		t.Fatalf("prepareWorkspace failed for C code: %v", err)
	}
	defer cleanup()

	if !strings.HasSuffix(cmd, "/solution") {
		t.Errorf("Expected command to end with '/solution', got %q", cmd)
	}
	if len(args) != 0 {
		t.Errorf("Expected empty args for compiled binary, got %v", args)
	}
}

func TestPrepareWorkspaceCPPSuccess(t *testing.T) {
	req := ExecuteRequest{
		Language: "cpp",
		Code:     "#include <iostream>\nint main() { std::cout << \"Hello CPP\" << std::endl; return 0; }",
	}

	cmd, args, cleanup, err := prepareWorkspace(&req, "test-exec-cpp")
	if err != nil {
		t.Fatalf("prepareWorkspace failed for CPP code: %v", err)
	}
	defer cleanup()

	if !strings.HasSuffix(cmd, "/solution") {
		t.Errorf("Expected command to end with '/solution', got %q", cmd)
	}
	if len(args) != 0 {
		t.Errorf("Expected empty args for compiled binary, got %v", args)
	}
}

func TestPrepareWorkspaceCCompilationError(t *testing.T) {
	req := ExecuteRequest{
		Language: "c",
		Code:     "#include <stdio.h>\nint main() { printf(\"syntax error\") return 0; }",
	}

	_, _, cleanup, err := prepareWorkspace(&req, "test-exec-c-err")
	if cleanup != nil {
		defer cleanup()
	}

	if err == nil {
		t.Fatalf("Expected compilation error, but got nil")
	}

	compErr, ok := err.(*CompilationError)
	if !ok {
		t.Fatalf("Expected *CompilationError, got %T: %v", err, err)
	}

	if compErr.Stderr == "" {
		t.Errorf("Expected non-empty Stderr in CompilationError")
	}
}

func TestExecuteHandlerCCompilationErrorResponse(t *testing.T) {
	cleanup := setupConfigWithMode("interpreter")
	defer cleanup()

	reqBody, _ := json.Marshal(ExecuteRequest{
		Language: "c",
		Code:     "#include <stdio.h>\nint main() { invalid_syntax_here }",
	})

	req, err := http.NewRequest(http.MethodPost, "/execute", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleExecute)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("Expected HTTP 200 for compilation error response, got: %v", status)
	}

	var resp ExecuteResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if resp.Result != "failed" {
		t.Errorf("Expected Result 'failed', got %q", resp.Result)
	}
	if resp.ErrorType != "compilation_error" {
		t.Errorf("Expected ErrorType 'compilation_error', got %q", resp.ErrorType)
	}
	if resp.Execution == nil || resp.Execution.Stderr == "" {
		t.Errorf("Expected Execution.Stderr to contain compiler error details")
	}
}
