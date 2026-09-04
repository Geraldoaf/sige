package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sige/internal/api/handlers"
	"sige/internal/api/middleware"
	"sige/internal/api/presenter"
	"sige/internal/auth"
	"sige/internal/config"
	"sige/internal/sandbox"
)

const (
	maxPayloadBytes = 10 << 20 // 10MB
	defaultStateDir = "/var/lib/sige"
)

var defaultKeyProvider = auth.NewFileKeyProvider(stateDir(), os.Getenv("SIGE_ALLOW_UNAUTHENTICATED") == "true")

// stateDir retorna o diretório de estado do servidor.
func stateDir() string {
	if d := strings.TrimSpace(os.Getenv("SIGE_STATE_DIR")); d != "" {
		return d
	}
	return defaultStateDir
}

// loadAPIKey retorna a chave ativa delegando para o KeyProvider.
func loadAPIKey() string {
	return defaultKeyProvider.ResolveKey()
}

// announceAPIKey exibe no log a forma de autenticação ativa no início do servidor.
func announceAPIKey() {
	key := defaultKeyProvider.ResolveKey()

	if key == "" {
		if os.Getenv("SIGE_ALLOW_UNAUTHENTICATED") == "true" {
			log.Println("[SIGE] Autenticação desabilitada (SIGE_ALLOW_UNAUTHENTICATED=true)")
		} else {
			log.Println("[SIGE] AVISO: nenhuma chave de API configurada.")
		}
		return
	}

	if defaultKeyProvider.IsGenerated() {
		fmt.Fprintf(os.Stdout, "[SIGE] API Key gerada: %s\n", key)
	} else {
		masked := key
		if len(key) >= 8 {
			masked = key[:4] + "..." + key[len(key)-4:]
		}
		fmt.Fprintf(os.Stdout, "[SIGE] API Key carregada: %s\n", masked)
	}
}

// trustedProxies retorna os IPs de proxies confiáveis a partir de SIGE_TRUSTED_PROXIES.
func trustedProxies() []string {
	raw := os.Getenv("SIGE_TRUSTED_PROXIES")
	if raw == "" {
		return nil
	}
	var proxies []string
	for _, p := range strings.Split(raw, ",") {
		if t := strings.TrimSpace(p); t != "" {
			proxies = append(proxies, t)
		}
	}
	return proxies
}

// Bridges para retrocompatibilidade com server_extended_test.go:
func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	mw := middleware.SecurityHeaders(next)
	return mw.ServeHTTP
}

func realIP(r *http.Request, proxies []string) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}
	for _, proxy := range proxies {
		if remoteIP == strings.TrimSpace(proxy) {
			if ip := r.Header.Get("X-Real-IP"); ip != "" {
				return ip
			}
		}
	}
	return remoteIP
}

func rateLimitMiddleware(proxies []string) func(http.HandlerFunc) http.HandlerFunc {
	limiter := middleware.NewIPRateLimiter(10, 20, proxies)
	return func(next http.HandlerFunc) http.HandlerFunc {
		mw := limiter.Middleware()(next)
		return mw.ServeHTTP
	}
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	mw := middleware.AuthMiddleware(defaultKeyProvider)(next)
	return mw.ServeHTTP
}

// StartServer inicia o servidor HTTP e configura o encadeamento de middlewares com graceful shutdown.
func StartServer(port string) {
	announceAPIKey()

	trusted := trustedProxies()
	rateLimiter := middleware.NewIPRateLimiter(10, 20, trusted)
	defer rateLimiter.Close()

	authMW := middleware.AuthMiddleware(defaultKeyProvider)

	mux := http.NewServeMux()

	// Rotas Públicas
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/health", handlers.HandleHealth)
	mux.HandleFunc("/ready", handlers.HandleReady)

	// Rotas Autenticadas
	mux.Handle("/languages", authMW(http.HandlerFunc(handlers.HandleLanguages)))
	mux.Handle("/capacity", authMW(http.HandlerFunc(handlers.HandleCapacity)))
	mux.Handle("/metrics", authMW(http.HandlerFunc(handlers.HandleMetrics)))
	mux.Handle("/validate", authMW(http.HandlerFunc(handlers.HandleValidate)))
	mux.Handle("/execute", authMW(http.HandlerFunc(handleExecute)))

	rootHandler := middleware.SecurityHeaders(rateLimiter.Middleware()(mux))

	addr := port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           rootHandler,
		ReadHeaderTimeout: 3 * time.Second,  // Proteção contra ataques Slowloris
		ReadTimeout:       10 * time.Second, // Timeout de leitura de body
		WriteTimeout:      35 * time.Second, // Timeout suficiente para cobrir SIGE_CEILING_TIMEOUT_SEC
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 1)
	go func() {
		log.Printf("[SIGE] Servidor HTTP ouvindo em %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		log.Fatalf("[SIGE] Erro fatal no servidor HTTP: %v", err)
	case <-ctx.Done():
		log.Println("[SIGE] Sinal de desligamento recebido. Encerrando servidor graciosamente...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[SIGE] Erro no encerramento: %v", err)
		}
	}
}

// HandleHome expõe o handler da rota raiz para testes.
func HandleHome(w http.ResponseWriter, r *http.Request) {
	handleHome(w, r)
}

// HandleExecute expõe o handler de execução para testes.
func HandleExecute(w http.ResponseWriter, r *http.Request) {
	handleExecute(w, r)
}

// handleHome processa requisições para a rota raiz (compatibilidade).
func handleHome(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "API SIGE - Running in API mode")
}

// handleExecute valida a requisição, prepara o workspace e executa o código no sandbox.
func handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		presenter.RenderError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "HTTP method not allowed", nil)
		return
	}

	// Limite de payload: rejeita bodies maiores que 10MB
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadBytes)

	defaultCfg, err := config.Resolve("config.json")
	if err != nil {
		presenter.RenderError(w, http.StatusInternalServerError, "CONFIG_ERROR", "Server error: invalid configuration: "+err.Error(), nil)
		return
	}

	var req ExecuteRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			presenter.RenderError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request payload too large (max 10MB)", nil)
			return
		}
		presenter.RenderError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request JSON: "+err.Error(), nil)
		return
	}

	apiMode := strings.ToLower(strings.TrimSpace(defaultCfg.APIMode))
	if apiMode == "" {
		apiMode = "interpreter"
	}
	if err := validateRequest(&req, apiMode); err != nil {
		presenter.RenderError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	resolvedMem, resolvedCPU, resolvedTimeout, resolvedTmpLimit, resolvedFileSize, resolvedOpenFiles := resolveLimits(&req, &defaultCfg, apiMode)

	execID := fmt.Sprintf("exec-%d-%d", time.Now().UnixNano(), os.Getpid())
	command, args, workspace, cleanup, err := prepareWorkspace(&req, execID)
	if err != nil {
		if errors.Is(err, sandbox.ErrAtCapacity) {
			w.Header().Set("Retry-After", "30")
			presenter.RenderError(w, http.StatusServiceUnavailable, "AT_CAPACITY", "Server at capacity: too many sandboxes running, retry shortly.", nil)
			return
		}

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
			presenter.RenderJSON(w, http.StatusOK, response)
			handlers.RecordExecution("compilation_error")
			return
		}
		presenter.RenderError(w, http.StatusInternalServerError, "WORKSPACE_ERROR", "Error preparing temporary workspace: "+err.Error(), nil)
		return
	}
	defer cleanup()

	runSandbox := func(stdinInput string) (sandbox.ExecutionResult, error) {
		cfg := sandbox.Config{
			Name:          fmt.Sprintf("sandbox-%s-%d", execID, time.Now().UnixNano()%100000),
			Workspace:     workspace,
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

	status := "success"
	if response.Execution != nil && response.Execution.Status != "" {
		status = response.Execution.Status
	}
	handlers.RecordExecution(status)

	presenter.RenderJSON(w, http.StatusOK, response)
}
