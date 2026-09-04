package presenter

import (
	"encoding/json"
	"net/http"
)

// RenderJSON renderiza qualquer payload Go em formato JSON com código de status HTTP especificado.
func RenderJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
