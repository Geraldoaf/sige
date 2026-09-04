package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"sige/internal/api/middleware"
	"testing"
)

type mockKeyProvider struct {
	validKey string
}

func (m *mockKeyProvider) ResolveKey() string { return m.validKey }
func (m *mockKeyProvider) Validate(key string) bool { return key == m.validKey }
func (m *mockKeyProvider) IsGenerated() bool { return false }

func TestAuthMiddleware(t *testing.T) {
	provider := &mockKeyProvider{validKey: "secret-123"}
	authMW := middleware.AuthMiddleware(provider)

	handler := authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Sem header -> 401
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 para requisição sem chave, obtido %d", rr1.Code)
	}

	// 2. Com chave inválida -> 401
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("X-API-Key", "invalid-key")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401 para chave inválida, obtido %d", rr2.Code)
	}

	// 3. Com chave correta -> 200
	req3 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req3.Header.Set("X-API-Key", "secret-123")
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Errorf("esperado 200 para chave correta, obtido %d", rr3.Code)
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if ct := rr.Header().Get("X-Content-Type-Options"); ct != "nosniff" {
		t.Errorf("esperado X-Content-Type-Options nosniff, obtido %s", ct)
	}
	if fo := rr.Header().Get("X-Frame-Options"); fo != "DENY" {
		t.Errorf("esperado X-Frame-Options DENY, obtido %s", fo)
	}
}
