// Package handlers provides HTTP request handlers for KB-6 Formulary Service.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

// writeErrorResponse writes a JSON error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:     http.StatusText(statusCode),
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// writeJSONResponse writes a JSON response with proper headers
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Timestamp", time.Now().UTC().Format(time.RFC3339))
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Fallback to writing error
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// parseIntParam parses an integer query parameter with a default value
func parseIntParam(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return defaultValue
	}
	return parsed
}

// parseFloatParam parses a float query parameter with a default value
func parseFloatParam(value string, defaultValue float64) float64 {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

// getStringParam gets a string query parameter or returns empty string
func getStringParam(r *http.Request, key string, aliases ...string) string {
	value := r.URL.Query().Get(key)
	if value != "" {
		return value
	}
	for _, alias := range aliases {
		value = r.URL.Query().Get(alias)
		if value != "" {
			return value
		}
	}
	return ""
}

// getOptionalString returns a pointer to string if non-empty, nil otherwise
func getOptionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// getBoolParam parses a boolean query parameter
func getBoolParam(r *http.Request, key string) bool {
	value := r.URL.Query().Get(key)
	return value == "true" || value == "1" || value == "yes"
}
