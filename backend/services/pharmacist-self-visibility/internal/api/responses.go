package api

import (
	"encoding/json"
	"net/http"
)

// ErrorEnvelope is the standard JSON error body returned by all endpoints.
type ErrorEnvelope struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// WriteJSON serialises body to JSON and writes it with the given HTTP status.
// Content-Type is set to application/json automatically.
func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// WriteError writes a standard JSON error response.
func WriteError(w http.ResponseWriter, status int, code, msg string) {
	WriteJSON(w, status, ErrorEnvelope{Error: msg, Code: code})
}
