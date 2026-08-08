package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sige/internal/config"
	"sige/internal/sandbox"
	"strings"
	"time"
)

func StartServer(port string) {
	fmt.Printf("API Server started at http://localhost:%s\n", port)

	http.HandleFunc("/execute", HandleExecute)
	http.HandleFunc("/", HandleHome)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting HTTP server: %v\n", err)
		os.Exit(1)
	}
}

func HandleHome(w http.ResponseWriter, r *http.Request) {
	handleHome(w, r)
}

func HandleExecute(w http.ResponseWriter, r *http.Request) {
	handleExecute(w, r)
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "API SIGE - Running in API mode")
}

func handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "HTTP method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defaultCfg, err := config.LoadConfig("config.json")
	if err != nil {
		http.Error(w, "Server error: configuration file 'config.json' not found in root directory.", http.StatusInternalServerError)
		return
	}

	var req ExecuteRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, "Invalid request JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	apiMode := strings.ToLower(strings.TrimSpace(defaultCfg.APIMode))
	if apiMode == "" {
		apiMode = "interpreter"
	}
	if err := validateRequest(&req, apiMode); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resolvedMem, resolvedCPU, resolvedTimeout, resolvedTmpLimit, resolvedFileSize, resolvedOpenFiles := resolveLimits(&req, &defaultCfg, apiMode)

	execID := fmt.Sprintf("exec-%d-%d", time.Now().UnixNano(), os.Getpid())
	command, args, cleanup, err := prepareWorkspace(&req, execID)
	if err != nil {
		var compErr *CompilationError
		if errors.As(err, &compErr) {
			resResult := "failed"
			if apiMode == "single_evaluation" || apiMode == "multi_evaluation" {
				resResult = "FAIL"
			}
			total := 1
			if apiMode == "multi_evaluation" {
				total = len(req.TestCases)
			}
			response := ExecuteResponse{
				Mode:        apiMode,
				Result:      resResult,
				ErrorType:   "compilation_error",
				PassedCount: 0,
				TotalCount:  total,
				Execution: &GraderExecution{
					Stderr: compErr.Stderr,
					Status: "compilation_error",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "Error preparing temporary workspace: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer cleanup()

	runSandbox := func(stdinInput string) (sandbox.ExecutionResult, error) {
		cfg := sandbox.Config{
			Name:          fmt.Sprintf("sandbox-%s-%d", execID, time.Now().UnixNano()%100000),
			MemoryMB:      resolvedMem,
			CPU:           resolvedCPU,
			TimeoutSec:    resolvedTimeout,
			TmpLimitMB:    resolvedTmpLimit,
			MaxFileSizeMB: resolvedFileSize,
			MaxOpenFiles:  resolvedOpenFiles,
			Stdin:         stdinInput,
		}
		return sandbox.Execute(cfg, command, args)
	}

	var response ExecuteResponse
	switch apiMode {
	case "interpreter":
		response = runInterpreter(runSandbox, req.Stdin)
	case "single_evaluation":
		response = runSingleEvaluation(runSandbox, req.Stdin, req.ExpectedStdout)
	case "multi_evaluation":
		response = runMultiEvaluation(runSandbox, req.TestCases)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
