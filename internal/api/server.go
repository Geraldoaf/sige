package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sige/internal/config"
	"sige/internal/sandbox"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// maxPayloadBytes define o limite de tamanho do corpo da requisição (10MB).
const maxPayloadBytes = 10 * 1024 * 1024

const defaultStateDir = "/var/lib/sige"

var (
	apiKeyOnce     sync.Once
	resolvedAPIKey string
)

// loadAPIKey obtém a API key configurada (Docker Secret, SIGE_API_KEY ou chave persistida/gerada).
func loadAPIKey() string {
	apiKeyOnce.Do(func() {
		if data, err := os.ReadFile("/run/secrets/sige_api_key"); err == nil {
			if k := strings.TrimSpace(string(data)); k != "" {
				resolvedAPIKey = k
				return
			}
		}
		if k := strings.TrimSpace(os.Getenv("SIGE_API_KEY")); k != "" {
			resolvedAPIKey = k
			return
		}
		if os.Getenv("SIGE_ALLOW_UNAUTHENTICATED") == "true" {
			return
		}
		resolvedAPIKey = loadOrCreatePersistedKey()
	})
	return resolvedAPIKey
}

// stateDir retorna o diretório de estado do servidor.
func stateDir() string {
	if d := strings.TrimSpace(os.Getenv("SIGE_STATE_DIR")); d != "" {
		return d
	}
	return defaultStateDir
}

// loadOrCreatePersistedKey lê a chave salva em disco ou gera uma nova aleatória de 32 bytes.
// apiKeyFoiGerada indica que a chave em uso nasceu NESTE start, e não foi
// lida de uma execução anterior. Só nesse caso ela é impressa por inteiro no
// log (ver announceAPIKey).
var apiKeyFoiGerada bool

func loadOrCreatePersistedKey() string {
	path := filepath.Join(stateDir(), "api_key")

	if data, err := os.ReadFile(path); err == nil {
		if k := strings.TrimSpace(string(data)); k != "" {
			return k
		}
	}

	apiKeyFoiGerada = true

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		fmt.Fprintf(os.Stderr, "[SIGE] Erro fatal ao gerar API key: %v\n", err)
		os.Exit(1)
	}
	key := hex.EncodeToString(buf)

	if err := os.MkdirAll(stateDir(), 0700); err == nil {
		if err := os.WriteFile(path, []byte(key+"\n"), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "[SIGE] Aviso: não foi possível persistir a API key em %s (%v)\n", path, err)
		}
	}

	return key
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

// realIP extrai o IP do cliente respeitando X-Real-IP apenas para proxies confiáveis.
func realIP(r *http.Request, proxies []string) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	for _, proxy := range proxies {
		if remoteIP == proxy {
			if ip := r.Header.Get("X-Real-IP"); ip != "" {
				return ip
			}
		}
	}
	return remoteIP
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	limiters   = make(map[string]*ipLimiter)
	limitersMu sync.Mutex
)

const (
	rateLimitPerSecond = 10
	rateLimitBurst     = 20
	limiterIdleTTL     = 5 * time.Minute
	limiterSweepEvery  = 1 * time.Minute
)

// getLimiter obtém ou cria o rate limiter para um endereço IP.
func getLimiter(ip string) *rate.Limiter {
	limitersMu.Lock()
	defer limitersMu.Unlock()

	if l, ok := limiters[ip]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}
	l := &ipLimiter{
		limiter:  rate.NewLimiter(rate.Limit(rateLimitPerSecond), rateLimitBurst),
		lastSeen: time.Now(),
	}
	limiters[ip] = l
	return l.limiter
}

// startLimiterCleaner executa uma goroutine para limpar rate limiters inativos periodicamente.
func startLimiterCleaner() {
	go func() {
		ticker := time.NewTicker(limiterSweepEvery)
		defer ticker.Stop()
		for range ticker.C {
			limitersMu.Lock()
			for k, v := range limiters {
				if time.Since(v.lastSeen) > limiterIdleTTL {
					delete(limiters, k)
				}
			}
			limitersMu.Unlock()
		}
	}()
}

// securityHeaders adiciona cabeçalhos HTTP básicos de segurança.
func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next(w, r)
	}
}

// authMiddleware valida o cabeçalho X-API-Key contra a chave esperada.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := loadAPIKey()
		if expected == "" {
			if os.Getenv("SIGE_ALLOW_UNAUTHENTICATED") == "true" {
				next(w, r)
				return
			}
			http.Error(w, "Server misconfigured: no API key configured (set SIGE_API_KEY, mount the sige_api_key secret, or set SIGE_ALLOW_UNAUTHENTICATED=true for local development)", http.StatusServiceUnavailable)
			return
		}
		provided := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// rateLimitMiddleware limita a taxa de requisições por IP de origem.
func rateLimitMiddleware(proxies []string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r, proxies)
			if !getLimiter(ip).Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next(w, r)
		}
	}
}

// chain encadeia múltiplos middlewares a um handler HTTP.
func chain(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// announceAPIKey exibe no log a forma de autenticação ativa no início do servidor.
func announceAPIKey() {
	key := loadAPIKey()

	if key == "" {
		fmt.Fprintln(os.Stderr, "[SIGE] AVISO: autenticação DESABILITADA (SIGE_ALLOW_UNAUTHENTICATED=true).")
		return
	}

	if os.Getenv("SIGE_API_KEY") != "" {
		fmt.Fprintln(os.Stderr, "[SIGE] Autenticação ativa — chave lida de SIGE_API_KEY.")
		return
	}
	if _, err := os.Stat("/run/secrets/sige_api_key"); err == nil {
		fmt.Fprintln(os.Stderr, "[SIGE] Autenticação ativa — chave lida do Docker Secret.")
		return
	}

	caminho := filepath.Join(stateDir(), "api_key")

	// A chave completa é impressa UMA única vez: no start em que ela foi
	// gerada. Nos reinícios seguintes sai apenas o prefixo.
	//
	// Sem essa distinção, todo restart reemitia a credencial para o stderr —
	// e, num ambiente com coleta centralizada de logs, ela ficaria
	// indefinidamente em texto claro num sistema cujo controle de acesso é
	// tipicamente mais frouxo que o do servidor. Imprimir na geração é o que
	// permite subir sem configuração prévia; repetir a cada boot não tem
	// utilidade e só amplia a exposição.
	fmt.Fprintln(os.Stderr, "[SIGE] ────────────────────────────────────────────────────────────")
	if apiKeyFoiGerada {
		fmt.Fprintln(os.Stderr, "[SIGE] Nenhuma API key configurada; uma foi gerada automaticamente.")
		fmt.Fprintf(os.Stderr, "[SIGE]   X-API-Key: %s\n", key)
		fmt.Fprintln(os.Stderr, "[SIGE]   Anote agora: esta é a única vez que ela aparece no log.")
	} else {
		fmt.Fprintf(os.Stderr, "[SIGE] Autenticação ativa — usando a chave gerada anteriormente (%s…).\n", key[:8])
		fmt.Fprintf(os.Stderr, "[SIGE]   Para recuperá-la: docker compose exec sige-api cat %s\n", caminho)
	}
	fmt.Fprintf(os.Stderr, "[SIGE]   (guardada em %s)\n", caminho)
	fmt.Fprintln(os.Stderr, "[SIGE] Defina SIGE_API_KEY ou um Docker Secret para fixá-la.")
	fmt.Fprintln(os.Stderr, "[SIGE] ────────────────────────────────────────────────────────────")
}

const (
	httpReadHeaderTimeout = 10 * time.Second
	httpReadTimeout       = 60 * time.Second
	httpWriteTimeout      = 240 * time.Second
	httpIdleTimeout       = 60 * time.Second
	httpMaxHeaderBytes    = 1 << 16
)

// StartServer inicializa o servidor HTTP na porta especificada.
func StartServer(port string) {
	fmt.Fprintf(os.Stderr, "[SIGE] Servidor iniciado em http://localhost:%s\n", port)

	announceAPIKey()

	proxies := trustedProxies()
	rl := rateLimitMiddleware(proxies)
	startLimiterCleaner()

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", chain(handleExecute, securityHeaders, authMiddleware, rl))
	mux.HandleFunc("/", chain(handleHome, securityHeaders))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		ReadTimeout:       httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
		MaxHeaderBytes:    httpMaxHeaderBytes,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting HTTP server: %v\n", err)
		os.Exit(1)
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

// handleHome processa requisições para a rota raiz (healthcheck / status).
func handleHome(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "API SIGE - Running in API mode")
}

// handleExecute valida a requisição, prepara o workspace e executa o código no sandbox.
func handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "HTTP method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limite de payload (A1): rejeita bodies maiores que 10MB
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadBytes)

	// config.json é opcional: Resolve parte dos padrões embutidos, sobrepõe o
	// arquivo se ele existir e, por último, as variáveis SIGE_*.
	defaultCfg, err := config.Resolve("config.json")
	if err != nil {
		http.Error(w, "Server error: invalid configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var req ExecuteRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			http.Error(w, "Request payload too large (max 10MB)", http.StatusRequestEntityTooLarge)
			return
		}
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
	command, args, workspace, cleanup, err := prepareWorkspace(&req, execID)
	if err != nil {
		// Saturação na etapa de compilação é condição transitória do
		// servidor, não erro do cliente: 503 com Retry-After.
		if errors.Is(err, sandbox.ErrAtCapacity) {
			w.Header().Set("Retry-After", "30")
			http.Error(w, "Server at capacity: too many sandboxes running, retry shortly.", http.StatusServiceUnavailable)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
