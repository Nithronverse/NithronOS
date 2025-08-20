package server

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/disks"
	"nithronos/backend/nosd/pkg/auth"
)

func Logger(cfg config.Config) *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	logger := log.Logger.Level(cfg.LogLevel).With().Timestamp().Logger()
	return &logger
}

func NewRouter(cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(zerologMiddleware(Logger(cfg)))

	// Dev CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	r.Use(c.Handler)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "version": "0.1.0"})
	})

	// Auth
	store, _ := auth.NewStore(cfg.UsersPath)
	codec := auth.NewSessionCodec(cfg.SessionHashKey, cfg.SessionBlockKey)

	r.Post("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email, Password, Totp string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		u, err := store.GetByEmail(body.Email)
		if err != nil || !auth.VerifyPassword(auth.DefaultParams, u.PasswordHash, body.Password) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_credentials"})
			return
		}
		if u.TOTPSecret != "" && body.Totp == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"need_totp": true})
			return
		}
		if u.TOTPSecret != "" && body.Totp != "" && !auth.VerifyTOTP(u.TOTPSecret, body.Totp) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_totp"})
			return
		}
		sess := auth.Session{UserID: u.ID, Roles: u.Roles, TwoFA: u.TOTPSecret == "" || body.Totp != ""}
		_ = codec.EncodeToCookie(w, sess)
		csrf := auth.IssueCSRF(w)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "csrf": csrf})
	})

	r.Post("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		auth.ClearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		if s, ok := codec.DecodeFromRequest(r); ok {
			writeJSON(w, map[string]any{"id": s.UserID, "roles": s.Roles, "totp_enabled": s.TwoFA})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})

	// TOTP setup & confirm
	r.Post("/api/auth/totp/setup", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email, Password string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		u, err := store.GetByEmail(body.Email)
		if err != nil || !auth.VerifyPassword(auth.DefaultParams, u.PasswordHash, body.Password) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if u.TOTPSecret != "" {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "totp_already_enabled"})
			return
		}
		secret, uri, err := auth.GenerateTOTPSecret("NithronOS", u.Email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		u.TOTPSecret = secret
		_ = store.UpdateUser(u)
		writeJSON(w, map[string]any{"secret": secret, "otpauth": uri})
	})

	r.Post("/api/auth/totp/confirm", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Email, Code string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		u, err := store.GetByEmail(body.Email)
		if err != nil || u.TOTPSecret == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !auth.VerifyTOTP(u.TOTPSecret, body.Code) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	// Protected routes
	r.Group(func(pr chi.Router) {
		pr.Use(func(next http.Handler) http.Handler { return withUser(next, codec) })
		pr.Use(func(next http.Handler) http.Handler { return requireAuth(next, codec) })
		pr.Use(requireCSRF)

		pr.Get("/api/disks", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if runtime.GOOS != "windows" && hasCommand("lsblk") {
				if list, err := disks.Collect(ctx); err == nil {
					// Enrich with SMART when possible
					for i := range list {
						if list[i].Path != "" {
							list[i].Smart = disks.SmartSummaryFor(ctx, list[i].Path)
						}
					}
					writeJSON(w, map[string]any{"disks": list})
					return
				}
			}
			// Mock fallback
			writeJSON(w, map[string]any{"disks": []map[string]any{
				{"name": "sda", "kname": "sda", "path": "/dev/sda", "size": 1000204886016, "rota": true, "type": "disk", "tran": "sata", "vendor": "Mock", "model": "Disk A", "serial": "MOCKA123"},
				{"name": "nvme0n1", "kname": "nvme0n1", "path": "/dev/nvme0n1", "size": 512110190592, "rota": false, "type": "disk", "tran": "nvme", "vendor": "Mock", "model": "NVMe 512G", "serial": "MOCKNVME"},
			}})
		})

		pr.Get("/api/pools", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{})
		})

		pr.Get("/api/shares", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{})
		})

		pr.Get("/api/apps", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []map[string]string{{"id": "jellyfin", "status": "not_installed"}})
		})

		pr.Get("/api/remote/status", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"mode": "lan-only", "https": true})
		})
	})

	return r
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
