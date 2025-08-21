package server

import (
	"net/http"
	"strings"
)

// securityHeaders adds common security headers to every response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CSP: conservative defaults for API responses and simple HTML
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; img-src 'self' data:; object-src 'none'")
		// HSTS only when HTTPS (native TLS or trusted proxy header)
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		next.ServeHTTP(w, r)
	})
}
