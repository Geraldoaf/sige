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
					"Servidor malconfigurado: nenhuma chave de API foi fornecida ou configurada", nil)
				return
			}

			providedKey := r.Header.Get("X-API-Key")
			if !provider.Validate(providedKey) {
				presenter.RenderError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Não autorizado: chave X-API-Key ausente ou inválida", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
