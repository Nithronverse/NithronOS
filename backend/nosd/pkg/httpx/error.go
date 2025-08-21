package httpx

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type ErrorPayload struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	RetryAfterSec int    `json:"retryAfterSec,omitempty"`
}

// WriteError writes a JSON error response with a consistent shape:
// {"error": {"code":"...","message":"..."}}
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": ErrorPayload{Code: http.StatusText(statusCode), Message: message}})
}

// WriteTypedError writes a JSON error with explicit code and optional retryAfterSec.
func WriteTypedError(w http.ResponseWriter, statusCode int, code, message string, retryAfter int) {
	w.Header().Set("Content-Type", "application/json")
	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	}
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": ErrorPayload{Code: code, Message: message, RetryAfterSec: retryAfter}})
}
