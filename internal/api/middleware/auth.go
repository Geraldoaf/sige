package middleware

import (
	"net/http"
	"sige/internal/api/presenter"
	"sige/internal/auth"
)

// AuthMiddleware valida o cabeçalho X-API-Key contra o KeyProvider fornecido.
func AuthMiddleware(provider auth.KeyProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			expectedKey := provider.ResolveKey()
			if expectedKey == "" && !provider.Validate("") {
				presenter.RenderError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE",
					"Server misconfigured: no API key provided or configured", nil)
				return
			}

			providedKey := r.Header.Get("X-API-Key")
			if !provider.Validate(providedKey) {
				presenter.RenderError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized: missing or invalid X-API-Key header", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
