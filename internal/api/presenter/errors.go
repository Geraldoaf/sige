package presenter

import (
	"encoding/json"
	"net/http"
)

// APIError padroniza o corpo de erro de todas as respostas HTTP da API.
type APIError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// ErrorEnvelope envelopa APIError em um objeto { "error": ... }.
type ErrorEnvelope struct {
	Error APIError `json:"error"`
}

// RenderError renderiza uma resposta de erro em formato JSON estruturado.
func RenderError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorEnvelope{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}
