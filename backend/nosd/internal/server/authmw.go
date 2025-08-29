package server

import (
	"context"
	"net/http"

	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/pkg/auth"
	"nithronos/backend/nosd/pkg/httpx"
)

type ctxKey string

const (
	ctxUserID ctxKey = "uid"
	ctxRoles  ctxKey = "roles"
)

func withUser(next http.Handler, codec *auth.SessionCodec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, ok := codec.DecodeFromRequest(r); ok {
			r = r.WithContext(context.WithValue(r.Context(), ctxUserID, s.UserID))
			r = r.WithContext(context.WithValue(r.Context(), ctxRoles, s.Role))
		}
		next.ServeHTTP(w, r)
	})
}

func requireAuth(next http.Handler, codec *auth.SessionCodec, cfg config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if uid, ok := decodeSessionUID(r, cfg); ok && uid != "" {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := codec.DecodeFromRequest(r); ok {
			next.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
}

func requireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		ck, err := r.Cookie(auth.CSRFCookieName)
		if err != nil {
			httpx.WriteTypedError(w, http.StatusUnauthorized, "auth.csrf.missing", "Missing CSRF token", 0)
			return
		}
		if r.Header.Get("X-CSRF-Token") != ck.Value {
			httpx.WriteTypedError(w, http.StatusUnauthorized, "auth.csrf.invalid", "Invalid CSRF token", 0)
			return
		}
		next.ServeHTTP(w, r)
	})
}
