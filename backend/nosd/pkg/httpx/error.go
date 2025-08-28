package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type ErrorPayload struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	RetryAfterSec int    `json:"retryAfterSec,omitempty"`
	Details       any    `json:"details,omitempty"`
}

// WriteError writes a JSON error response with a consistent shape:
// {"error": {"code":"...","message":"..."}}
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": ErrorPayload{Code: http.StatusText(statusCode), Message: message}}); err != nil {
		fmt.Printf("Failed to write error response: %v\n", err)
	}
}

// WriteTypedError writes a JSON error with explicit code and optional retryAfterSec.
func WriteTypedError(w http.ResponseWriter, statusCode int, code, message string, retryAfter int) {
	w.Header().Set("Content-Type", "application/json")
	if retryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	}
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": ErrorPayload{Code: code, Message: message, RetryAfterSec: retryAfter}}); err != nil {
		fmt.Printf("Failed to write error response: %v\n", err)
	}
}

// WriteErrorWithDetails writes a JSON error with a stable code and additional details map.
func WriteErrorWithDetails(w http.ResponseWriter, statusCode int, code, message string, details map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": ErrorPayload{Code: code, Message: message, Details: details}}); err != nil {
		fmt.Printf("Failed to write error response: %v\n", err)
	}
}
