package api

import (
	"encoding/base64"
	"fmt"
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

func TestValidateRequest_Files_Valid(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("int soma(int a, int b) { return a + b; }"))
	req := &ExecuteRequest{
		Language: "c",
		Files: []FileEntry{
			{Name: "main.c", Content: "#include <stdio.h>\nint main(){return 0;}"},
			{Name: "include/calc.h", Content: "int soma(int a, int b);"},
			{Name: "src/calc.c", ContentBase64: b64},
		},
	}

	if err := validateRequest(req, "interpreter"); err != nil {
		t.Errorf("expected valid Files to pass validation, got %v", err)
	}
}

func TestValidateRequest_Files_UnsafePath(t *testing.T) {
	unsafePaths := []string{"../escape.c", "/root/bad.c", "src/../main.c", ".."}
	for _, p := range unsafePaths {
		req := &ExecuteRequest{
			Language: "c",
			Files: []FileEntry{
				{Name: p, Content: "int x = 1;"},
			},
		}
		if err := validateRequest(req, "interpreter"); err == nil {
			t.Errorf("expected error for unsafe path %q, got nil", p)
		}
	}
}

func TestValidateRequest_Files_InvalidBase64(t *testing.T) {
	req := &ExecuteRequest{
		Language: "c",
		Files: []FileEntry{
			{Name: "calc.c", ContentBase64: "invalid-base64!@#"},
		},
	}
	if err := validateRequest(req, "interpreter"); err == nil {
		t.Errorf("expected error for invalid base64 in files, got nil")
	}
}

func TestValidateRequest_Files_TooManyFiles(t *testing.T) {
	files := make([]FileEntry, maxFilesCount+1)
	for i := range files {
		files[i] = FileEntry{Name: fmt.Sprintf("file_%d.c", i), Content: "int x = 0;"}
	}
	req := &ExecuteRequest{
		Language: "c",
		Files:    files,
	}
	if err := validateRequest(req, "interpreter"); err == nil || !strings.Contains(err.Error(), "Too many files") {
		t.Errorf("expected error about exceeding maxFilesCount, got %v", err)
	}
}

func TestValidateRequest_Files_TotalSizeExceeded(t *testing.T) {
	chunk := strings.Repeat("a", (maxTotalFilesBytes/2)+100)
	req := &ExecuteRequest{
		Language: "python",
		Files: []FileEntry{
			{Name: "part1.py", Content: chunk},
			{Name: "part2.py", Content: chunk},
		},
	}
	if err := validateRequest(req, "interpreter"); err == nil || !strings.Contains(err.Error(), "Total size of 'files' is too large") {
		t.Errorf("expected error about exceeding maxTotalFilesBytes, got %v", err)
	}
}

func TestValidateRequest_Files_EmptyContent(t *testing.T) {
	req := &ExecuteRequest{
		Language: "python",
		Files: []FileEntry{
			{Name: "main.py", Content: "", ContentBase64: ""},
		},
	}
	if err := validateRequest(req, "interpreter"); err == nil || !strings.Contains(err.Error(), "must contain 'content' or 'content_base64'") {
		t.Errorf("expected error about empty content in files, got %v", err)
	}
}

func TestValidateRequest_Files_DuplicateAndCollisions(t *testing.T) {
	// Duplicate filenames
	reqDup := &ExecuteRequest{
		Language: "c",
		Files: []FileEntry{
			{Name: "main.c", Content: "int main(){return 0;}"},
			{Name: "main.c", Content: "int main(){return 0;}"},
		},
	}
	if err := validateRequest(reqDup, "interpreter"); err == nil || !strings.Contains(err.Error(), "Duplicate file path") {
		t.Errorf("expected error for duplicate file path, got %v", err)
	}

	// File and directory collision
	reqColl := &ExecuteRequest{
		Language: "python",
		Files: []FileEntry{
			{Name: "pkg", Content: "x = 1"},
			{Name: "pkg/mod.py", Content: "y = 2"},
		},
	}
	if err := validateRequest(reqColl, "interpreter"); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Errorf("expected error for file/directory collision, got %v", err)
	}
}

func TestValidateRequest_Files_ReservedBinaryName(t *testing.T) {
	for _, reserved := range []string{"solution", "solution/main.c"} {
		req := &ExecuteRequest{
			Language: "c",
			Files: []FileEntry{
				{Name: reserved, Content: "int main(){return 0;}"},
				{Name: "helper.c", Content: "int h(){return 1;}"},
			},
		}
		if err := validateRequest(req, "interpreter"); err == nil || !strings.Contains(err.Error(), "reserved binary name 'solution'") {
			t.Errorf("expected error for reserved name %q, got %v", reserved, err)
		}
	}
}

func TestValidateRequest_Files_HeaderOnlyRejected(t *testing.T) {
	req := &ExecuteRequest{
		Language: "cpp",
		Files: []FileEntry{
			{Name: "include/calc.hpp", Content: "int add(int a, int b);"},
		},
	}
	if err := validateRequest(req, "interpreter"); err == nil || !strings.Contains(err.Error(), "require at least one source file") {
		t.Errorf("expected error for header-only C++ submission, got %v", err)
	}
}

func TestValidateRequest_Files_CustomEntrypointInSubdir(t *testing.T) {
	reqValid := &ExecuteRequest{
		Language: "python",
		Filename: "src/app.py",
		Files: []FileEntry{
			{Name: "lib/util.py", Content: "X = 42"},
			{Name: "src/app.py", Content: "from lib.util import X\nprint(X)"},
		},
	}
	if err := validateRequest(reqValid, "interpreter"); err != nil {
		t.Errorf("expected custom relative entrypoint in 'files' to pass, got %v", err)
	}

	reqMissing := &ExecuteRequest{
		Language: "python",
		Filename: "src/nonexistent.py",
		Files: []FileEntry{
			{Name: "src/app.py", Content: "print(1)"},
		},
	}
	if err := validateRequest(reqMissing, "interpreter"); err == nil || !strings.Contains(err.Error(), "was not found in 'files'") {
		t.Errorf("expected error for missing entrypoint in 'files', got %v", err)
	}
}

func TestValidateRequest_Files_LeadingHyphenOptionInjection(t *testing.T) {
	for _, badName := range []string{"-fplugin=/tmp/pwn.so.c", "-E.c", "src/-O0.c"} {
		req := &ExecuteRequest{
			Language: "c",
			Files: []FileEntry{
				{Name: badName, Content: "int main(){return 0;}"},
			},
		}
		if err := validateRequest(req, "interpreter"); err == nil {
			t.Errorf("expected error for leading hyphen filename %q, got nil", badName)
		}
	}
}

func TestValidateRequest_CompileFlags(t *testing.T) {
	// 1. Valid flags with C
	reqValid := &ExecuteRequest{
		Language:     "c",
		Code:         "int main(){return 0;}",
		CompileFlags: CompileFlags{"-O3", "-std=c11", "-Wall", "-DDEBUG", "-lm"},
	}
	if err := validateRequest(reqValid, "interpreter"); err != nil {
		t.Errorf("expected valid compile_flags to pass, got: %v", err)
	}

	// 2. Flags passed via CompilerFlags alias
	reqAlias := &ExecuteRequest{
		Language:      "cpp",
		Code:          "int main(){return 0;}",
		CompilerFlags: CompileFlags{"-std=c++17", "-O2"},
	}
	if err := validateRequest(reqAlias, "interpreter"); err != nil {
		t.Errorf("expected compiler_flags alias to pass, got: %v", err)
	}

	// 3. Flags passed via Flags alias
	reqFlags := &ExecuteRequest{
		Language: "c",
		Code:     "int main(){return 0;}",
		Flags:    CompileFlags{"-O0"},
	}
	if err := validateRequest(reqFlags, "interpreter"); err != nil {
		t.Errorf("expected flags alias to pass, got: %v", err)
	}

	// 4. Rejected for Python
	reqPy := &ExecuteRequest{
		Language:     "python",
		Code:         "print(1)",
		CompileFlags: CompileFlags{"-O3"},
	}
	if err := validateRequest(reqPy, "interpreter"); err == nil || !strings.Contains(err.Error(), "only supported for compiled languages") {
		t.Errorf("expected error rejecting compile_flags for python, got: %v", err)
	}

	// 5. Rejected for -o output override
	reqOutput := &ExecuteRequest{
		Language:     "c",
		Code:         "int main(){return 0;}",
		CompileFlags: CompileFlags{"-o", "evil"},
	}
	if err := validateRequest(reqOutput, "interpreter"); err == nil || !strings.Contains(err.Error(), "cannot override output binary") {
		t.Errorf("expected error rejecting -o flag, got: %v", err)
	}

	// 6. Rejected for -c non-executable flag
	reqCompileOnly := &ExecuteRequest{
		Language:     "c",
		Code:         "int main(){return 0;}",
		CompileFlags: CompileFlags{"-c"},
	}
	if err := validateRequest(reqCompileOnly, "interpreter"); err == nil || !strings.Contains(err.Error(), "cannot include '-c'") {
		t.Errorf("expected error rejecting -c flag, got: %v", err)
	}
}
