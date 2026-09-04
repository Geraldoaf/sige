package api

import (
	"strings"
	"testing"
)

func TestValidateRequest_LanguageRequired(t *testing.T) {
	req := &ExecuteRequest{
		Language: "",
		Code:     "print('hello')",
	}
	err := validateRequest(req, "interpreter")
	if err == nil || !strings.Contains(err.Error(), "language") {
		t.Errorf("expected error about language, got %v", err)
	}
}

func TestValidateRequest_CodeOrBase64Required(t *testing.T) {
	req := &ExecuteRequest{
		Language: "python",
	}
	err := validateRequest(req, "interpreter")
	if err == nil || !strings.Contains(err.Error(), "code") {
		t.Errorf("expected error about missing code/file_base64, got %v", err)
	}
}

func TestValidateRequest_InvalidBase64(t *testing.T) {
	req := &ExecuteRequest{
		Language:   "python",
		FileBase64: "this-is-not-valid-base64!@#$%",
	}
	err := validateRequest(req, "interpreter")
	if err == nil || !strings.Contains(err.Error(), "not valid base64") {
		t.Errorf("expected error about invalid base64, got %v", err)
	}
}

func TestValidateRequest_ValidBase64(t *testing.T) {
	req := &ExecuteRequest{
		Language:   "python",
		FileBase64: "cHJpbnQoImhlbGxvIikK", // base64("print(\"hello\")\n")
	}
	err := validateRequest(req, "interpreter")
	if err != nil {
		t.Errorf("expected valid base64 to pass validation, got %v", err)
	}
}

func TestValidateRequest_CPUFormats(t *testing.T) {
	validCPUs := []string{"10", "10%", "50%", "100%", "10000 100000"}
	for _, cpu := range validCPUs {
		req := &ExecuteRequest{
			Language:       "python",
			Code:           "print('ok')",
			ExpectedStdout: "ok",
			CPU:            cpu,
		}
		if err := validateRequest(req, "single_evaluation"); err != nil {
			t.Errorf("expected valid CPU %q to pass validation, got %v", cpu, err)
		}
	}

	invalidCPUs := []string{"invalid", "50%%", "-10%", "10 20 30", "10,100", "cpu"}
	for _, cpu := range invalidCPUs {
		req := &ExecuteRequest{
			Language:       "python",
			Code:           "print('ok')",
			ExpectedStdout: "ok",
			CPU:            cpu,
		}
		if err := validateRequest(req, "single_evaluation"); err == nil {
			t.Errorf("expected invalid CPU %q to fail validation, got nil", cpu)
		}
	}
}

func TestValidateRequest_FilenameSecurity(t *testing.T) {
	safeNames := []string{"solution.py", "main.cpp", "test_123.sh", "code-v2.c", "script.py"}
	for _, name := range safeNames {
		req := &ExecuteRequest{
			Language: "python",
			Code:     "print(1)",
			Filename: name,
		}
		if err := validateRequest(req, "interpreter"); err != nil {
			t.Errorf("expected safe filename %q to pass validation, got %v", name, err)
		}
	}

	unsafeNames := []string{
		"../escape.py",
		"/etc/passwd",
		"sub/dir.py",
		"file;rm.py",
		"file|sh.py",
		"file`cmd`.py",
		"file$(cmd).py",
		".",
		"..",
		"file name.py", // spaces not allowed
	}
	for _, name := range unsafeNames {
		req := &ExecuteRequest{
			Language: "python",
			Code:     "print(1)",
			Filename: name,
		}
		if err := validateRequest(req, "interpreter"); err == nil {
			t.Errorf("expected unsafe filename %q to fail validation, got nil", name)
		}
	}
}

func TestValidateSizes_AllFields(t *testing.T) {
	// Exceed maxCodeBytes
	reqTooLargeCode := &ExecuteRequest{
		Language: "python",
		Code:     strings.Repeat("a", maxCodeBytes+1),
	}
	if err := validateSizes(reqTooLargeCode); err == nil {
		t.Errorf("expected error for code exceeding maxCodeBytes, got nil")
	}

	// Exceed maxFileBase64Bytes
	reqTooLargeBase64 := &ExecuteRequest{
		Language:   "python",
		FileBase64: strings.Repeat("a", maxFileBase64Bytes+1),
	}
	if err := validateSizes(reqTooLargeBase64); err == nil {
		t.Errorf("expected error for file_base64 exceeding maxFileBase64Bytes, got nil")
	}

	// Exceed maxFilenameBytes
	reqTooLongFilename := &ExecuteRequest{
		Language: "python",
		Code:     "print(1)",
		Filename: strings.Repeat("a", maxFilenameBytes+1),
	}
	if err := validateSizes(reqTooLongFilename); err == nil {
		t.Errorf("expected error for filename exceeding maxFilenameBytes, got nil")
	}

	// Exceed maxStdinBytes
	reqTooLargeStdin := &ExecuteRequest{
		Language: "python",
		Code:     "print(1)",
		Stdin:    strings.Repeat("a", maxStdinBytes+1),
	}
	if err := validateSizes(reqTooLargeStdin); err == nil {
		t.Errorf("expected error for stdin exceeding maxStdinBytes, got nil")
	}

	// Exceed maxExpectedStdoutBytes
	reqTooLargeStdout := &ExecuteRequest{
		Language:       "python",
		Code:           "print(1)",
		ExpectedStdout: strings.Repeat("a", maxExpectedStdoutBytes+1),
	}
	if err := validateSizes(reqTooLargeStdout); err == nil {
		t.Errorf("expected error for expected_stdout exceeding maxExpectedStdoutBytes, got nil")
	}

	// Exceed maxStdinBytes in test_cases
	reqTooLargeTestCaseStdin := &ExecuteRequest{
		Language: "python",
		Code:     "print(1)",
		TestCases: []TestCase{
			{Stdin: strings.Repeat("a", maxStdinBytes+1), ExpectedStdout: "ok"},
		},
	}
	if err := validateSizes(reqTooLargeTestCaseStdin); err == nil {
		t.Errorf("expected error for test_cases stdin exceeding limit, got nil")
	}

	// Exceed maxExpectedStdoutBytes in test_cases
	reqTooLargeTestCaseStdout := &ExecuteRequest{
		Language: "python",
		Code:     "print(1)",
		TestCases: []TestCase{
			{Stdin: "in", ExpectedStdout: strings.Repeat("a", maxExpectedStdoutBytes+1)},
		},
	}
	if err := validateSizes(reqTooLargeTestCaseStdout); err == nil {
		t.Errorf("expected error for test_cases expected_stdout exceeding limit, got nil")
	}
}

func TestValidateRequest_MultiEvaluationMaxTestCases(t *testing.T) {
	cases := make([]TestCase, maxTestCases+1)
	for i := range cases {
		cases[i] = TestCase{Stdin: "a", ExpectedStdout: "b"}
	}

	req := &ExecuteRequest{
		Language:  "python",
		Code:      "print(1)",
		TestCases: cases,
	}

	err := validateRequest(req, "multi_evaluation")
	if err == nil || !strings.Contains(err.Error(), "Too many test_cases") {
		t.Errorf("expected error about exceeding max test cases (20), got %v", err)
	}
}

func TestValidateRequest_UnknownAPIMode(t *testing.T) {
	req := &ExecuteRequest{
		Language: "python",
		Code:     "print(1)",
	}

	err := validateRequest(req, "invalid_mode")
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Errorf("expected error for unknown API mode, got %v", err)
	}
}
