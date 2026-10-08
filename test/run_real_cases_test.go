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
	sigeBin := "../sige"
	if _, err := os.Stat(sigeBin); os.IsNotExist(err) {
		if _, errOpt := os.Stat("/opt/sige/sige"); errOpt == nil {
			sigeBin = "/opt/sige/sige"
		} else {
			cmd := exec.Command("go", "build", "-o", sigeBin, "../main.go")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Erro ao compilar binário sige para testes CLI: %v", err)
			}
		}
	}

	outHelp, err := exec.Command(sigeBin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("Comando sige --help falhou: %v", err)
	}
	if !strings.Contains(string(outHelp), "Available Commands:") {
		t.Errorf("Saída do --help não contém a seção de comandos: %s", string(outHelp))
	}

	outCg, err := exec.Command(sigeBin, "cgroups-version").CombinedOutput()
	if err != nil {
		t.Fatalf("Comando sige cgroups-version falhou: %v", err)
	}
	if !strings.Contains(string(outCg), "cgroup version") {
		t.Errorf("Saída do cgroups-version inesperada: %s", string(outCg))
	}

	outRunTaskHelp, err := exec.Command(sigeBin, "run-task", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("Comando sige run-task --help falhou: %v", err)
	}
	if !strings.Contains(string(outRunTaskHelp), "cgroup v2 sandbox") {
		t.Errorf("Saída do run-task --help inesperada: %s", string(outRunTaskHelp))
	}
}

// TestRealCaseAPI_AllPayloadsSuite carrega e submete todos os 36 casos reais de test/real_cases via POST em /execute
func TestRealCaseAPI_AllPayloadsSuite(t *testing.T) {
	cleanup := setupConfig(t, "interpreter")
	defer cleanup()

	entries, err := os.ReadDir("real_cases")
	if err != nil {
		t.Fatalf("Erro ao listar diretório real_cases: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".py") &&
			!strings.HasSuffix(entry.Name(), ".c") &&
			!strings.HasSuffix(entry.Name(), ".cpp") &&
			!strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}

		filename := entry.Name()
		t.Run(filename, func(t *testing.T) {
			content, err := os.ReadFile("real_cases/" + filename)
			if err != nil {
				t.Fatalf("Erro ao ler arquivo %s: %v", filename, err)
			}

			var lang string
			switch {
			case strings.HasSuffix(filename, ".py"):
				lang = "python"
			case strings.HasSuffix(filename, ".c"):
				lang = "c"
			case strings.HasSuffix(filename, ".cpp"):
				lang = "cpp"
			case strings.HasSuffix(filename, ".sh"):
				lang = "bash"
			}

			reqBody, _ := json.Marshal(api.ExecuteRequest{
				Language: lang,
				Code:     string(content),
			})

			req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(reqBody))
			rr := httptest.NewRecorder()

			api.HandleExecute(rr, req)

			if rr.Code != http.StatusOK {
				t.Logf("[%s] API retornou HTTP %d: %s", filename, rr.Code, rr.Body.String())
			} else {
				var resp api.ExecuteResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err == nil {
					t.Logf("[%s] OK - Mode: %s, Result: %s, ErrorType: %s", filename, resp.Mode, resp.Result, resp.ErrorType)
					if filename == "42_forged_stderr_detection.c" {
						if resp.Execution != nil && resp.Execution.Status == "file_size_exceeded" {
							t.Errorf("[%s] Falha de segurança: o daemon caiu no ataque de stderr forjado e marcou file_size_exceeded!", filename)
						}
					}
				}
			}
		})
	}
}

// TestRealCaseAPI_MaliciousPostPayloads envia requisições POST com payloads maliciosos ou anômalos
func TestRealCaseAPI_MaliciousPostPayloads(t *testing.T) {
	cleanup := setupConfig(t, "interpreter")
	defer cleanup()

	tests := []struct {
		name       string
		payload    api.ExecuteRequest
		expectFail bool
		expectCode int
	}{
		{
			name: "Language With Spaces",
			payload: api.ExecuteRequest{
				Language: "  python  ",
				Code:     "print('spaced language')",
			},
			expectCode: http.StatusOK,
		},
		{
			name: "Unicode and Null Bytes in Stdin",
			payload: api.ExecuteRequest{
				Language: "python",
				Code:     "import sys\nprint('Got:', repr(sys.stdin.read()))",
				Stdin:    "Null:\x00 Unicode:\u2728\U0001F600",
			},
			expectCode: http.StatusOK,
		},
		{
			name: "Dangerous Shell Metachars in Filename",
			payload: api.ExecuteRequest{
				Language: "python",
				Code:     "print(1)",
				Filename: "test;rm -rf /;.py",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "DotDot Directory Traversal in Filename",
			payload: api.ExecuteRequest{
				Language: "python",
				Code:     "print(1)",
				Filename: "../../../evil.py",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Empty Code and Empty Base64",
			payload: api.ExecuteRequest{
				Language: "python",
				Code:     "",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Invalid Base64 in FileBase64",
			payload: api.ExecuteRequest{
				Language:   "python",
				FileBase64: "###notbase64@@@",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Path Traversal in Files Entry",
			payload: api.ExecuteRequest{
				Language: "python",
				Files: []api.FileEntry{
					{Name: "../../etc/evil.py", Content: "print(1)"},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Duplicate Filenames in Files",
			payload: api.ExecuteRequest{
				Language: "c",
				Files: []api.FileEntry{
					{Name: "main.c", Content: "int main(){return 0;}"},
					{Name: "main.c", Content: "int main(){return 1;}"},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "File and Directory Collision in Files",
			payload: api.ExecuteRequest{
				Language: "python",
				Files: []api.FileEntry{
					{Name: "pkg", Content: "x = 1"},
					{Name: "pkg/util.py", Content: "y = 2"},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Reserved Binary Name 'solution' in C Files",
			payload: api.ExecuteRequest{
				Language: "c",
				Files: []api.FileEntry{
					{Name: "solution", Content: "fake_elf"},
					{Name: "main.c", Content: "int main(){return 0;}"},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Header-Only C Submission Without Source File",
			payload: api.ExecuteRequest{
				Language: "c",
				Files: []api.FileEntry{
					{Name: "include/only_header.h", Content: "int x = 10;"},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compiler Flag Injection via Leading Hyphen in Files",
			payload: api.ExecuteRequest{
				Language: "c",
				Files: []api.FileEntry{
					{Name: "-fplugin=/tmp/evil.so.c", Content: "int main(){return 0;}"},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Non-Existent Entrypoint Filename with Files",
			payload: api.ExecuteRequest{
				Language: "python",
				Filename: "src/missing.py",
				Files: []api.FileEntry{
					{Name: "src/app.py", Content: "print('ok')"},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags Rejected for Python",
			payload: api.ExecuteRequest{
				Language:     "python",
				Code:         "print('ok')",
				CompileFlags: api.CompileFlags{"-O3"},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags Output Override in C",
			payload: api.ExecuteRequest{
				Language:     "c",
				Code:         "int main(){return 0;}",
				CompileFlags: api.CompileFlags{"-o", "evil"},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags Non-Executable Flag in C",
			payload: api.ExecuteRequest{
				Language:     "c",
				Code:         "int main(){return 0;}",
				CompileFlags: api.CompileFlags{"-c"},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags Response File @/etc/passwd",
			payload: api.ExecuteRequest{
				Language:     "c",
				Code:         "int main(){return 0;}",
				CompileFlags: api.CompileFlags{"@/etc/passwd"},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags GCC Plugin Injection -fplugin",
			payload: api.ExecuteRequest{
				Language:     "c",
				Code:         "int main(){return 0;}",
				CompileFlags: api.CompileFlags{"-fplugin=/tmp/evil.so"},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags Custom Specs Injection -specs",
			payload: api.ExecuteRequest{
				Language:     "c",
				Code:         "int main(){return 0;}",
				CompileFlags: api.CompileFlags{"-specs=/tmp/evil.spec"},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags Forced Include -include",
			payload: api.ExecuteRequest{
				Language:     "c",
				Code:         "int main(){return 0;}",
				CompileFlags: api.CompileFlags{"-include=/etc/passwd"},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "Compile Flags Dependency File Output -MF",
			payload: api.ExecuteRequest{
				Language:     "c",
				Code:         "int main(){return 0;}",
				CompileFlags: api.CompileFlags{"-MF=/workspace/solution"},
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
			rr := httptest.NewRecorder()

			api.HandleExecute(rr, req)

			if rr.Code != tt.expectCode {
				t.Errorf("[%s] Esperado status %d, obtido %d: %s", tt.name, tt.expectCode, rr.Code, rr.Body.String())
			}
		})
	}
}

// TestRealCaseAPI_MultiFiles_ExecutionAndCompilation testa o envio de múltiplos arquivos e compilação multi-módulo.
func TestRealCaseAPI_MultiFiles_ExecutionAndCompilation(t *testing.T) {
	cleanup := setupConfig(t, "single_evaluation")
	defer cleanup()

	t.Run("MultiFile Python Package Import", func(t *testing.T) {
		payload := api.ExecuteRequest{
			Language:       "python",
			Filename:       "src/app.py",
			ExpectedStdout: "RESULT=42\n",
			Files: []api.FileEntry{
				{
					Name:    "src/app.py",
					Content: "import sys, os\nsys.path.insert(0, '/workspace')\nfrom pkg.calc import somar\nprint(f'RESULT={somar(15, 27)}')\n",
				},
				{
					Name:    "pkg/calc.py",
					Content: "def somar(a, b):\n    return a + b\n",
				},
			},
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		api.HandleExecute(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("Esperado HTTP 200, obtido %d: %s", rr.Code, rr.Body.String())
		}
		var resp api.ExecuteResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err == nil {
			t.Logf("[MultiFile Python] Result: %s, ErrorType: %s, Stdout: %q, Stderr: %q", resp.Result, resp.ErrorType, resp.Execution.Stdout, resp.Execution.Stderr)
			if os.Geteuid() == 0 && resp.Result != "PASS" {
				t.Errorf("Esperado PASS no MultiFile Python, obtido %s (%s): %s", resp.Result, resp.ErrorType, resp.Execution.Stderr)
			}
		}
	})

	t.Run("MultiFile C Compilation with Header and Subdir", func(t *testing.T) {
		payload := api.ExecuteRequest{
			Language:       "c",
			ExpectedStdout: "SOMA=42\n",
			Files: []api.FileEntry{
				{
					Name:    "main.c",
					Content: "#include <stdio.h>\n#include \"include/math_op.h\"\nint main(void) {\n    printf(\"SOMA=%d\\n\", multiplicar(6, 7));\n    return 0;\n}\n",
				},
				{
					Name:    "include/math_op.h",
					Content: "#ifndef MATH_OP_H\n#define MATH_OP_H\nint multiplicar(int a, int b);\n#endif\n",
				},
				{
					Name:    "src/math_op.c",
					Content: "#include \"include/math_op.h\"\nint multiplicar(int a, int b) {\n    return a * b;\n}\n",
				},
			},
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		api.HandleExecute(rr, req)

		if rr.Code != http.StatusOK {
			if os.Geteuid() != 0 && rr.Code == http.StatusInternalServerError {
				t.Skipf("Pulando compilação C no sandbox: requer privilégios de root/cgroup (%s)", rr.Body.String())
			}
			t.Fatalf("Esperado HTTP 200, obtido %d: %s", rr.Code, rr.Body.String())
		}
		var resp api.ExecuteResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err == nil {
			t.Logf("[MultiFile C] Result: %s, ErrorType: %s, Stdout: %q, Stderr: %q", resp.Result, resp.ErrorType, resp.Execution.Stdout, resp.Execution.Stderr)
			if os.Geteuid() == 0 && resp.Result != "PASS" {
				t.Errorf("Esperado PASS no MultiFile C, obtido %s (%s): %s", resp.Result, resp.ErrorType, resp.Execution.Stderr)
			}
		}
	})

	t.Run("MultiFile C Project 48 (Multiple .c and .h files with math lib)", func(t *testing.T) {
		projectFiles := []string{
			"main.c",
			"include/geometry.h",
			"include/stats.h",
			"src/geometry.c",
			"src/stats.c",
		}
		var entries []api.FileEntry
		for _, rel := range projectFiles {
			data, err := os.ReadFile("real_cases/48_multifile_c_project/" + rel)
			if err != nil {
				t.Fatalf("Erro ao ler arquivo do projeto C multi-file (%s): %v", rel, err)
			}
			entries = append(entries, api.FileEntry{
				Name:    rel,
				Content: string(data),
			})
		}

		payload := api.ExecuteRequest{
			Language:       "c",
			CompileFlags:   api.CompileFlags{"-lm"},
			ExpectedStdout: "PERIMETER=12.00 MEAN=5.00 STDDEV=2.00 READONLY_OK\n",
			Files:          entries,
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		api.HandleExecute(rr, req)

		if rr.Code != http.StatusOK {
			if os.Geteuid() != 0 && rr.Code == http.StatusInternalServerError {
				t.Skipf("Pulando compilação do projeto C 48 no sandbox: requer privilégios de root/cgroup (%s)", rr.Body.String())
			}
			t.Fatalf("Esperado HTTP 200, obtido %d: %s", rr.Code, rr.Body.String())
		}
		var resp api.ExecuteResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err == nil {
			t.Logf("[MultiFile C Project 48] Result: %s, ErrorType: %s, Stdout: %q, Stderr: %q", resp.Result, resp.ErrorType, resp.Execution.Stdout, resp.Execution.Stderr)
			if os.Geteuid() == 0 && resp.Result != "PASS" {
				t.Errorf("Esperado PASS no MultiFile C Project 48, obtido %s (%s): %s", resp.Result, resp.ErrorType, resp.Execution.Stderr)
			}
		}
	})
}

// TestRealCaseAPI_CompileFlags_Execution testa a compilação com flags customizadas (array e string, aliases e -Werror).
func TestRealCaseAPI_CompileFlags_Execution(t *testing.T) {
	cleanup := setupConfig(t, "single_evaluation")
	defer cleanup()

	t.Run("C Macro Definitions and C11 Standard via Array", func(t *testing.T) {
		cCode := `#include <stdio.h>
#define STR_HELPER(x) #x
#define STR(x) STR_HELPER(x)
int main(void) {
    int x = 6;
    int type_check = _Generic(x, int: 1, default: 0);
    printf("BANNER=%s VAL=%d C11=%d\n", STR(BANNER_TOKEN), x * MULTIPLIER, type_check);
    return 0;
}
`
		payload := api.ExecuteRequest{
			Language:       "c",
			Code:           cCode,
			CompileFlags:   api.CompileFlags{"-O3", "-std=c11", "-DMULTIPLIER=7", "-DBANNER_TOKEN=SIGE_FLAGS_OK"},
			ExpectedStdout: "BANNER=SIGE_FLAGS_OK VAL=42 C11=1\n",
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		api.HandleExecute(rr, req)

		if rr.Code != http.StatusOK {
			if os.Geteuid() != 0 && rr.Code == http.StatusInternalServerError {
				t.Skipf("Pulando compilação C no sandbox: requer privilégios de root/cgroup (%s)", rr.Body.String())
			}
			t.Fatalf("Esperado HTTP 200, obtido %d: %s", rr.Code, rr.Body.String())
		}
		var resp api.ExecuteResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err == nil {
			t.Logf("[CompileFlags C Array] Result: %s, ErrorType: %s, Stdout: %q, Stderr: %q", resp.Result, resp.ErrorType, resp.Execution.Stdout, resp.Execution.Stderr)
			if os.Geteuid() == 0 && resp.Result != "PASS" {
				t.Errorf("Esperado PASS em C Macro Definitions, obtido %s (%s): %s", resp.Result, resp.ErrorType, resp.Execution.Stderr)
			}
		}
	})

	t.Run("CPP20 Standard and String Flags via compiler_flags Alias", func(t *testing.T) {
		rawJSON := `{
			"language": "cpp",
			"compiler_flags": "-std=c++20 -O2 -DCPP_BONUS=22",
			"expected_stdout": "CPP20=1 TOTAL=42\n",
			"code": "#include <iostream>\nint main() {\n    int is_cpp20 = (__cplusplus >= 202002L) ? 1 : 0;\n    std::cout << \"CPP20=\" << is_cpp20 << \" TOTAL=\" << (20 + CPP_BONUS) << \"\\n\";\n    return 0;\n}\n"
		}`

		req := httptest.NewRequest(http.MethodPost, "/execute", strings.NewReader(rawJSON))
		rr := httptest.NewRecorder()
		api.HandleExecute(rr, req)

		if rr.Code != http.StatusOK {
			if os.Geteuid() != 0 && rr.Code == http.StatusInternalServerError {
				t.Skipf("Pulando compilação C++20 no sandbox: requer privilégios de root/cgroup (%s)", rr.Body.String())
			}
			t.Fatalf("Esperado HTTP 200, obtido %d: %s", rr.Code, rr.Body.String())
		}
		var resp api.ExecuteResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err == nil {
			t.Logf("[CompileFlags CPP20 String] Result: %s, ErrorType: %s, Stdout: %q, Stderr: %q", resp.Result, resp.ErrorType, resp.Execution.Stdout, resp.Execution.Stderr)
			if os.Geteuid() == 0 && resp.Result != "PASS" {
				t.Errorf("Esperado PASS em CPP20 String Flags, obtido %s (%s): %s", resp.Result, resp.ErrorType, resp.Execution.Stderr)
			}
		}
	})

	t.Run("C Werror via flags Alias Triggers CompilationError", func(t *testing.T) {
		rawJSON := `{
			"language": "c",
			"flags": ["-Werror", "-Wunused-variable"],
			"expected_stdout": "OK\n",
			"code": "#include <stdio.h>\nint main(void) {\n    int unused_var = 123;\n    printf(\"OK\\n\");\n    return 0;\n}\n"
		}`

		req := httptest.NewRequest(http.MethodPost, "/execute", strings.NewReader(rawJSON))
		rr := httptest.NewRecorder()
		api.HandleExecute(rr, req)

		if rr.Code != http.StatusOK {
			if os.Geteuid() != 0 && rr.Code == http.StatusInternalServerError {
				t.Skipf("Pulando compilação C -Werror no sandbox: requer privilégios de root/cgroup (%s)", rr.Body.String())
			}
			t.Fatalf("Esperado HTTP 200, obtido %d: %s", rr.Code, rr.Body.String())
		}
		var resp api.ExecuteResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err == nil {
			t.Logf("[CompileFlags -Werror] Result: %s, ErrorType: %s, Stderr: %q", resp.Result, resp.ErrorType, resp.Execution.Stderr)
			if os.Geteuid() == 0 && (resp.Result != "FAIL" || resp.ErrorType != "compilation_error") {
				t.Errorf("Esperado FAIL (compilation_error) com -Werror, obtido Result=%s ErrorType=%s", resp.Result, resp.ErrorType)
			}
		}
	})
}
