package server

import (
	"context"
	"net/http"

	"nithronos/backend/nosd/pkg/auth"
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
			r = r.WithContext(context.WithValue(r.Context(), ctxRoles, s.Roles))
		}
		next.ServeHTTP(w, r)
	})
}

func requireAuth(next http.Handler, codec *auth.SessionCodec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := codec.DecodeFromRequest(r); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
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
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.Header.Get("X-CSRF-Token") != ck.Value {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
