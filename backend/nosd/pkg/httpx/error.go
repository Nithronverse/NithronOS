package httpx

import (
	"encoding/json"
	"net/http"
)

// WriteError writes a JSON error response with a consistent shape: {"error": "message"}.
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
