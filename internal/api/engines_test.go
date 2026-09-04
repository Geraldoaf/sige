package api

import (
	"errors"
	"sige/internal/sandbox"
	"strings"
	"testing"
	"time"
)

func TestRunInterpreter_Success(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		return sandbox.ExecutionResult{
			Stdout:           "Hello World\n",
			Stderr:           "",
			Status:           "success",
			ExitCode:         0,
			Duration:         15 * time.Millisecond,
			CPUUserTime:      5000 * time.Microsecond,
			CPUSystemTime:    2000 * time.Microsecond,
			MemoryPeak:       1024 * 1024,
			LimitMemoryBytes: 50 * 1024 * 1024,
		}, nil
	}

	resp := runInterpreter(mockSandbox, "")

	if resp.Mode != "interpreter" {
		t.Errorf("expected mode 'interpreter', got %q", resp.Mode)
	}
	if resp.Result != "completed" {
		t.Errorf("expected result 'completed', got %q", resp.Result)
	}
	if resp.PassedCount != 1 || resp.TotalCount != 1 {
		t.Errorf("expected 1/1 passed, got %d/%d", resp.PassedCount, resp.TotalCount)
	}
	if resp.Execution == nil || resp.Execution.Stdout != "Hello World\n" {
		t.Errorf("expected Execution.Stdout 'Hello World\\n', got %v", resp.Execution)
	}
}

func TestRunInterpreter_Failure(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		return sandbox.ExecutionResult{
			Stdout:   "",
			Stderr:   "Traceback (most recent call last):\nZeroDivisionError",
			Status:   "failed",
			ExitCode: 1,
		}, errors.New("exit status 1")
	}

	resp := runInterpreter(mockSandbox, "")

	if resp.Result != "failed" {
		t.Errorf("expected result 'failed', got %q", resp.Result)
	}
	if resp.PassedCount != 0 {
		t.Errorf("expected 0 passed, got %d", resp.PassedCount)
	}
	if resp.ErrorType != "failed" {
		t.Errorf("expected error_type 'failed', got %q", resp.ErrorType)
	}
}

func TestRunSingleEvaluation_Pass(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		return sandbox.ExecutionResult{
			Stdout:   "  42 \n",
			Status:   "success",
			ExitCode: 0,
		}, nil
	}

	resp := runSingleEvaluation(mockSandbox, "15 27", "42")

	if resp.Result != "PASS" {
		t.Errorf("expected result 'PASS', got %q", resp.Result)
	}
	if resp.PassedCount != 1 || resp.TotalCount != 1 {
		t.Errorf("expected 1/1 passed, got %d/%d", resp.PassedCount, resp.TotalCount)
	}
}

func TestRunSingleEvaluation_Mismatch(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		return sandbox.ExecutionResult{
			Stdout:   "100\n",
			Status:   "success",
			ExitCode: 0,
		}, nil
	}

	resp := runSingleEvaluation(mockSandbox, "15 27", "42")

	if resp.Result != "FAIL" {
		t.Errorf("expected result 'FAIL', got %q", resp.Result)
	}
	if resp.ErrorType != "output_mismatch" {
		t.Errorf("expected error_type 'output_mismatch', got %q", resp.ErrorType)
	}
	if resp.PassedCount != 0 {
		t.Errorf("expected 0 passed, got %d", resp.PassedCount)
	}
	if resp.Expected != "42" || resp.Actual != "100\n" {
		t.Errorf("expected Expected='42' and Actual='100\\n', got %q / %q", resp.Expected, resp.Actual)
	}
}

func TestRunSingleEvaluation_Timeout(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		return sandbox.ExecutionResult{
			Status:   "timeout",
			ExitCode: -1,
		}, errors.New("timeout")
	}

	resp := runSingleEvaluation(mockSandbox, "in", "out")

	if resp.Result != "FAIL" {
		t.Errorf("expected result 'FAIL', got %q", resp.Result)
	}
	if resp.ErrorType != "timeout" {
		t.Errorf("expected error_type 'timeout', got %q", resp.ErrorType)
	}
}

func TestRunMultiEvaluation_AllPass(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		out := "ODD"
		if strings.TrimSpace(stdin) == "4" {
			out = "EVEN"
		}
		return sandbox.ExecutionResult{
			Stdout:   out + "\n",
			Status:   "success",
			ExitCode: 0,
		}, nil
	}

	testCases := []TestCase{
		{Stdin: "4", ExpectedStdout: "EVEN"},
		{Stdin: "7", ExpectedStdout: "ODD"},
	}

	resp := runMultiEvaluation(mockSandbox, testCases)

	if resp.Result != "PASS" {
		t.Errorf("expected result 'PASS', got %q", resp.Result)
	}
	if resp.PassedCount != 2 || resp.TotalCount != 2 {
		t.Errorf("expected 2/2 passed, got %d/%d", resp.PassedCount, resp.TotalCount)
	}
	if resp.FailedTestIndex != nil {
		t.Errorf("expected nil FailedTestIndex, got %v", *resp.FailedTestIndex)
	}
}

func TestRunMultiEvaluation_SecondFails(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		if strings.TrimSpace(stdin) == "1" {
			return sandbox.ExecutionResult{Stdout: "OK\n", Status: "success"}, nil
		}
		return sandbox.ExecutionResult{Stdout: "WRONG\n", Status: "success"}, nil
	}

	testCases := []TestCase{
		{Stdin: "1", ExpectedStdout: "OK"},
		{Stdin: "2", ExpectedStdout: "EXPECTED_2"},
	}

	resp := runMultiEvaluation(mockSandbox, testCases)

	if resp.Result != "FAIL" {
		t.Errorf("expected result 'FAIL', got %q", resp.Result)
	}
	if resp.PassedCount != 1 || resp.TotalCount != 2 {
		t.Errorf("expected 1/2 passed, got %d/%d", resp.PassedCount, resp.TotalCount)
	}
	if resp.FailedTestIndex == nil || *resp.FailedTestIndex != 1 {
		t.Errorf("expected FailedTestIndex 1, got %v", resp.FailedTestIndex)
	}
	if resp.ErrorType != "output_mismatch" {
		t.Errorf("expected error_type 'output_mismatch', got %q", resp.ErrorType)
	}
	if resp.Expected != "EXPECTED_2" {
		t.Errorf("expected Expected='EXPECTED_2', got %q", resp.Expected)
	}
}

func TestRunMultiEvaluation_EmptyTestCases(t *testing.T) {
	mockSandbox := func(stdin string) (sandbox.ExecutionResult, error) {
		return sandbox.ExecutionResult{Status: "success"}, nil
	}

	resp := runMultiEvaluation(mockSandbox, []TestCase{})

	if resp.Result != "PASS" || resp.TotalCount != 0 {
		t.Errorf("expected PASS with TotalCount 0, got Result=%q TotalCount=%d", resp.Result, resp.TotalCount)
	}
}
