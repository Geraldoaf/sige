package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sige/internal/api"
	"sige/internal/sandbox"
	"testing"
)

// TestBugDiscovery_UnsupportedLanguageReturns500InsteadOf400 demonstra que
// a API aceita linguagens não suportadas no validateRequest (retorna 200/500 em vez de 400 Bad Request).
// Como validateRequest não valida se a linguagem é uma das suportadas (python, bash, c, cpp),
// a requisição chega no prepareWorkspace e estoura com 500 Internal Server Error.
func TestBugDiscovery_UnsupportedLanguageReturns500InsteadOf400(t *testing.T) {
	cleanup := setupConfig(t, "interpreter")
	defer cleanup()

	unsupportedLanguages := []string{"rust", "ruby", "java", "go", "php", "javascript", "pascal"}

	for _, lang := range unsupportedLanguages {
		t.Run("lang_"+lang, func(t *testing.T) {
			reqBody, _ := json.Marshal(api.ExecuteRequest{
				Language: lang,
				Code:     "fn main() {}",
			})

			req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(reqBody))
			rr := httptest.NewRecorder()

			api.HandleExecute(rr, req)

			// O comportamento esperado da API para entradas inválidas do cliente é 400 Bad Request.
			// No entanto, o sistema atual retorna 500 Internal Server Error devido à falta de validação em validateRequest.
			if rr.Code == http.StatusInternalServerError {
				t.Logf("[BUG DETECTADO] Para linguagem não suportada %q, a API retornou status %d (Internal Server Error) em vez de 400 (Bad Request): %s",
					lang, rr.Code, rr.Body.String())
			} else if rr.Code != http.StatusBadRequest {
				t.Errorf("Esperado status 400 Bad Request para linguagem não suportada %q, obtido %d", lang, rr.Code)
			}
		})
	}
}

// TestBugDiscovery_NegativeCPUInSandboxConfig demonstra que Config.GetFormattedCPU
// aceita porcentagens negativas como "-50%" e gera valores negativos de quota ("-50000 100000"),
// o que é inválido para o cgroups v2.
func TestBugDiscovery_NegativeCPUInSandboxConfig(t *testing.T) {
	cfg := sandbox.Config{
		CPU: "-50%",
	}

	formatted := cfg.GetFormattedCPU()
	if formatted == "-50000 100000" {
		t.Logf("[EDGE CASE DETECTADO] GetFormattedCPU(\"-50%%\") retornou cgroup quota negativa inválida: %q (deveria retornar \"\" ou sanitizar)", formatted)
	}
}

// TestBugDiscovery_FilenameStartingWithDash demonstra que isSafeFilename permite
// nomes de arquivos iniciados com traço, por exemplo "-Wall.py" ou "-rf.c", o que pode
// colidir com flags de linha de comando dos compiladores/interpretadores.
func TestBugDiscovery_FilenameStartingWithDash(t *testing.T) {
	reqBody, _ := json.Marshal(api.ExecuteRequest{
		Language: "python",
		Code:     "print(1)",
		Filename: "-flag.py",
	})

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	api.HandleExecute(rr, req)

	// Se o filename foi aceito, loga o comportamento
	t.Logf("[POTENCIAL VULNERABILIDADE / EDGE CASE] Filename iniciado com hífen '-flag.py' foi processado com status: %d", rr.Code)
}

// TestBugDiscovery_WhitespaceOnlyCode demonstra o comportamento quando o código é apenas espaços em branco.
func TestBugDiscovery_WhitespaceOnlyCode(t *testing.T) {
	reqBody, _ := json.Marshal(api.ExecuteRequest{
		Language: "python",
		Code:     "    \n\t  \n  ",
	})

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	api.HandleExecute(rr, req)
	t.Logf("[BEHAVIOR CHECK] Código apenas com espaços em branco retornou status: %d", rr.Code)
}
