package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := securityHeaders(dummyHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %q", got)
	}
	if got := rr.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got %q", got)
	}
}

func TestRealIP_DirectConnection(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/execute", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	req.Header.Set("X-Real-IP", "10.0.0.1")

	// No trusted proxies configured
	ip := realIP(req, nil)
	if ip != "192.168.1.50" {
		t.Errorf("expected direct IP 192.168.1.50 ignoring untrusted X-Real-IP, got %q", ip)
	}
}

func TestRealIP_TrustedProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/execute", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.195")

	proxies := []string{"10.0.0.1", "127.0.0.1"}
	ip := realIP(req, proxies)
	if ip != "203.0.113.195" {
		t.Errorf("expected X-Real-IP 203.0.113.195 from trusted proxy, got %q", ip)
	}
}

func TestRateLimitMiddleware_ExceedBurst(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rl := rateLimitMiddleware(nil)(dummyHandler)

	clientIP := "198.51.100.22:54321"

	// Rate limit burst is 20
	rejected := false
	for i := 0; i < 30; i++ {
		req := httptest.NewRequest(http.MethodPost, "/execute", nil)
		req.RemoteAddr = clientIP
		rr := httptest.NewRecorder()

		rl.ServeHTTP(rr, req)

		if rr.Code == http.StatusTooManyRequests {
			rejected = true
			break
		}
	}

	if !rejected {
		t.Errorf("expected rate limiter to reject requests exceeding burst limit (20), but none returned 429")
	}
}

func TestExecuteHandler_PayloadTooLarge(t *testing.T) {
	cleanup := setupConfigWithMode("interpreter")
	defer cleanup()

	// Create payload larger than 10MB
	largeCode := strings.Repeat("A", maxPayloadBytes+1024)
	reqBody, _ := json.Marshal(ExecuteRequest{
		Language: "python",
		Code:     largeCode,
	})

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(reqBody))
	rr := httptest.NewRecorder()

	HandleExecute(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge && rr.Code != http.StatusBadRequest {
		t.Errorf("expected 413 or 400 for payload exceeding 10MB, got status %d: %s", rr.Code, rr.Body.String())
	}
}
