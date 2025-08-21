package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"nithronos/backend/nosd/handlers"
	"nithronos/backend/nosd/internal/apps"
	pwhash "nithronos/backend/nosd/internal/auth/hash"
	userstore "nithronos/backend/nosd/internal/auth/store"
	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/disks"
	"nithronos/backend/nosd/internal/pools"
	"nithronos/backend/nosd/internal/shares"
	"nithronos/backend/nosd/pkg/agentclient"
	"nithronos/backend/nosd/pkg/auth"
	"nithronos/backend/nosd/pkg/firewall"
	"nithronos/backend/nosd/pkg/httpx"
	poolroots "nithronos/backend/nosd/pkg/pools"
	"nithronos/backend/nosd/pkg/snapdb"

	"github.com/gorilla/securecookie"
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

	// CORS: strict to UI origin; allow credentials
	allowed := []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	if ui := os.Getenv("NOS_UI_ORIGIN"); ui != "" {
		allowed = []string{ui}
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   allowed,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-CSRF-Token", "Authorization"},
		AllowCredentials: true,
	})
	r.Use(c.Handler)

	// Init auth stores and session codec (reused across handlers)
	store, _ := auth.NewStore(cfg.UsersPath)
	users, _ := userstore.New(cfg.UsersPath)
	codec := auth.NewSessionCodec(cfg.SessionHashKey, cfg.SessionBlockKey)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "version": "0.1.0"})
	})

	// Setup routes mounted only while firstBoot=true
	r.Route("/api/setup", func(sr chi.Router) {
		sr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if users != nil && users.HasAdmin() {
					w.WriteHeader(http.StatusGone)
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		sr.Get("/state", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"firstBoot": true, "otpRequired": true})
		})

		// Rate limiter (very simple, in-memory per-IP)
		rl := newRateLimiter(5, time.Minute)
		sr.Post("/verify-otp", func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if h := r.Header.Get("X-Forwarded-For"); h != "" {
				ip = strings.Split(h, ",")[0]
			}
			if !rl.Allow(ip) {
				httpx.WriteError(w, http.StatusTooManyRequests, "rate limited")
				return
			}
			var body struct{ OTP string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.OTP) != 6 {
				httpx.WriteError(w, http.StatusBadRequest, "otp required")
				return
			}
			var st struct {
				OTP       string `json:"otp"`
				CreatedAt string `json:"created_at"`
				Used      bool   `json:"used"`
			}
			if b, err := os.ReadFile(cfg.FirstBootPath); err == nil {
				_ = json.Unmarshal(b, &st)
			}
			if st.Used || st.OTP == "" || st.OTP != body.OTP {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid otp")
				return
			}
			if t, err := time.Parse(time.RFC3339, st.CreatedAt); err != nil || time.Since(t) >= 15*time.Minute {
				httpx.WriteError(w, http.StatusUnauthorized, "otp expired")
				return
			}
			// Mark used immediately
			st.Used = true
			if b2, err := json.MarshalIndent(st, "", "  "); err == nil {
				_ = os.WriteFile(cfg.FirstBootPath, b2, 0o600)
			}
			payload := map[string]any{"purpose": "setup", "exp": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)}
			val, err := setupEncodeToken(cfg, payload)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "token error")
				return
			}
			writeJSON(w, map[string]any{"ok": true, "token": val})
		})

		sr.Post("/create-admin", func(w http.ResponseWriter, r *http.Request) {
			if users == nil {
				httpx.WriteError(w, http.StatusInternalServerError, "user store unavailable")
				return
			}
			authz := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(authz, prefix) {
				httpx.WriteError(w, http.StatusUnauthorized, "missing bearer")
				return
			}
			tok := strings.TrimSpace(authz[len(prefix):])
			claims, err := setupDecodeToken(cfg, tok)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			if claims["purpose"] != "setup" {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			if expStr, ok := claims["exp"].(string); ok {
				if t, err := time.Parse(time.RFC3339, expStr); err != nil || time.Now().After(t) {
					httpx.WriteError(w, http.StatusUnauthorized, "token expired")
					return
				}
			} else {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			var body struct {
				Username   string `json:"username"`
				Password   string `json:"password"`
				EnableTOTP bool   `json:"enable_totp"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			uname := strings.TrimSpace(body.Username)
			if !validUsername(uname) {
				httpx.WriteError(w, http.StatusBadRequest, "invalid username")
				return
			}
			if !validPassword(body.Password) {
				httpx.WriteError(w, http.StatusBadRequest, "weak password")
				return
			}
			if _, err := users.FindByUsername(uname); err == nil {
				httpx.WriteError(w, http.StatusConflict, "username taken")
				return
			}
			phc, err := pwhash.HashPassword(body.Password)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "hash error")
				return
			}
			now := time.Now().UTC().Format(time.RFC3339)
			u := userstore.User{ID: generateUUID(), Username: uname, PasswordHash: phc, Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now}
			if body.EnableTOTP {
				u.TOTPEnc = "pending"
			}
			if err := users.UpsertUser(u); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "persist error")
				return
			}
			writeJSON(w, map[string]any{"ok": true})
		})
	})

	// Login-specific limiters: per-IP 5/15m; per-username 10/15m
	loginIPRL := newRateLimiter(5, 15*time.Minute)
	loginUserRL := newRateLimiter(10, 15*time.Minute)

	// First admin creation (consumes first-boot OTP)
	r.Post("/api/setup/first-admin", func(w http.ResponseWriter, r *http.Request) {
		if users == nil {
			httpx.WriteError(w, http.StatusInternalServerError, "user store unavailable")
			return
		}
		if users.HasAdmin() {
			httpx.WriteError(w, http.StatusConflict, "admin already exists")
			return
		}
		var body struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			OTP      string `json:"otp"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		uname := strings.TrimSpace(body.Username)
		if uname == "" {
			uname = strings.TrimSpace(body.Email)
		}
		if uname == "" || strings.TrimSpace(body.Password) == "" || len(body.OTP) != 6 {
			httpx.WriteError(w, http.StatusBadRequest, "username, password and 6-digit otp required")
			return
		}
		// Load first-boot OTP state
		var st struct {
			OTP       string `json:"otp"`
			CreatedAt string `json:"created_at"`
			Used      bool   `json:"used"`
		}
		if b, err := os.ReadFile(cfg.FirstBootPath); err == nil {
			_ = json.Unmarshal(b, &st)
		}
		if st.Used || st.OTP == "" || st.OTP != body.OTP {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid otp")
			return
		}
		if t, err := time.Parse(time.RFC3339, st.CreatedAt); err != nil || time.Since(t) >= 15*time.Minute {
			httpx.WriteError(w, http.StatusUnauthorized, "otp expired")
			return
		}
		// Create admin user
		phc, err := pwhash.HashPassword(body.Password)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "hash error")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		u := userstore.User{ID: generateUUID(), Username: uname, PasswordHash: phc, Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now}
		if err := users.UpsertUser(u); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "persist error")
			return
		}
		// Mark OTP used (best-effort)
		st.Used = true
		_ = os.WriteFile(cfg.FirstBootPath, mustJSON(st), 0o600)
		writeJSON(w, map[string]any{"ok": true})
	})

	// Auth (legacy + new store integration)

	r.Post("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			Code       string `json:"code"`
			RememberMe bool   `json:"rememberMe"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		uname := strings.TrimSpace(body.Username)
		pass := body.Password
		// rate limit by IP and username
		ip := r.RemoteAddr
		if h := r.Header.Get("X-Forwarded-For"); h != "" {
			ip = strings.Split(h, ",")[0]
		}
		if !loginIPRL.Allow("ip:"+ip) || (uname != "" && !loginUserRL.Allow("u:"+uname)) {
			httpx.WriteError(w, http.StatusTooManyRequests, "try again later")
			return
		}
		u, err := users.FindByUsername(uname)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Check account lock
		if u.LockedUntil != "" {
			if t, err := time.Parse(time.RFC3339, u.LockedUntil); err == nil && time.Now().Before(t) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		ph := u.PasswordHash
		ok := false
		if strings.HasPrefix(ph, "dev:") || strings.HasPrefix(ph, "plain:") {
			ok = strings.TrimPrefix(strings.TrimPrefix(ph, "dev:"), "plain:") == pass
		} else {
			ok = pwhash.VerifyPassword(ph, pass)
		}
		if !ok {
			// increment failure; lock after 10
			u.FailedAttempts++
			if u.FailedAttempts >= 10 {
				u.FailedAttempts = 0
				u.LockedUntil = time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
			}
			_ = users.UpsertUser(u)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// If TOTP set, require code or recovery
		if u.TOTPEnc != "" || len(u.RecoveryHashes) > 0 {
			code := strings.TrimSpace(body.Code)
			if code == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "code required")
				return
			}
			verified := false
			if len(code) == 6 {
				if sec, err := decryptWithSecretKey(cfg.SecretPath, u.TOTPEnc); err == nil && auth.VerifyTOTP(string(sec), code) {
					verified = true
				}
			}
			if !verified {
				h := hashRecovery(code)
				for i, rh := range u.RecoveryHashes {
					if rh == h {
						verified = true
						// consume
						u.RecoveryHashes = append(u.RecoveryHashes[:i], u.RecoveryHashes[i+1:]...)
						_ = users.UpsertUser(u)
						break
					}
				}
			}
			if !verified {
				u.FailedAttempts++
				if u.FailedAttempts >= 10 {
					u.FailedAttempts = 0
					u.LockedUntil = time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)
				}
				_ = users.UpsertUser(u)
				httpx.WriteError(w, http.StatusUnauthorized, "invalid code")
				return
			}
		}
		// success: reset counters
		u.FailedAttempts = 0
		u.LockedUntil = ""
		_ = users.UpsertUser(u)
		if err := issueSessionCookies(w, cfg, u.ID, body.RememberMe); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "session error")
			return
		}
		issueCSRFCookie(w)
		writeJSON(w, map[string]any{"ok": true})
	})

	r.Post("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) { clearAuthCookies(w); w.WriteHeader(http.StatusNoContent) })

	r.Get("/api/auth/me", func(w http.ResponseWriter, r *http.Request) {
		if uid, ok := decodeSessionUID(r, cfg); ok {
			if u, err := users.FindByID(uid); err == nil {
				writeJSON(w, map[string]any{"user": map[string]any{"id": u.ID, "username": u.Username, "roles": u.Roles}})
				return
			}
		}
		if s, ok := codec.DecodeFromRequest(r); ok {
			writeJSON(w, map[string]any{"user": map[string]any{"id": s.UserID, "roles": s.Roles}})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})

	r.Post("/api/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if uid, ok := decodeRefreshUID(r, cfg); ok {
			if err := issueSessionCookies(w, cfg, uid, true); err == nil {
				writeJSON(w, map[string]any{"ok": true})
				return
			}
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
		// Require auth via new opaque cookies or legacy session cookie
		authRequired := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if _, ok := decodeSessionUID(r, cfg); ok {
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
		pr.Use(authRequired)
		pr.Use(requireCSRF)

		// AdminRequired middleware: resolve current user and assert role
		adminRequired := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				uid, ok := decodeSessionUID(r, cfg)
				if !ok {
					if s, ok2 := codec.DecodeFromRequest(r); ok2 {
						uid = s.UserID
						ok = true
					}
				}
				if !ok || uid == "" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				u, err := users.FindByID(uid)
				if err != nil {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				isAdmin := false
				for _, r := range u.Roles {
					if r == "admin" {
						isAdmin = true
						break
					}
				}
				if !isAdmin {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			})
		}

		// TOTP enroll (logged-in): generate secret, encrypt with secret.key, store pending enc
		pr.Get("/api/auth/totp/enroll", func(w http.ResponseWriter, r *http.Request) {
			uid, ok := decodeSessionUID(r, cfg)
			if !ok {
				if s, ok2 := codec.DecodeFromRequest(r); ok2 {
					uid = s.UserID
					ok = true
				}
			}
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			u, err := users.FindByID(uid)
			if err != nil {
				httpx.WriteError(w, http.StatusNotFound, "user not found")
				return
			}
			secret, uri, err := auth.GenerateTOTPSecret("NithronOS", u.Username)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "totp error")
				return
			}
			enc, err := encryptWithSecretKey(cfg.SecretPath, []byte(secret))
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "encrypt error")
				return
			}
			u.TOTPEnc = enc
			if err := users.UpsertUser(u); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "persist error")
				return
			}
			writeJSON(w, map[string]any{"otpauth_url": uri, "qr_png_base64": ""})
		})

		// TOTP verify (logged-in): verify code, generate recovery codes and persist hashes
		pr.Post("/api/auth/totp/verify", func(w http.ResponseWriter, r *http.Request) {
			uid, ok := decodeSessionUID(r, cfg)
			if !ok {
				if s, ok2 := codec.DecodeFromRequest(r); ok2 {
					uid = s.UserID
					ok = true
				}
			}
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			u, err := users.FindByID(uid)
			if err != nil {
				httpx.WriteError(w, http.StatusNotFound, "user not found")
				return
			}
			var body struct{ Code string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.Code) != 6 {
				httpx.WriteError(w, http.StatusBadRequest, "invalid code")
				return
			}
			secretB, err := decryptWithSecretKey(cfg.SecretPath, u.TOTPEnc)
			if err != nil {
				httpx.WriteError(w, http.StatusBadRequest, "invalid state")
				return
			}
			if !auth.VerifyTOTP(string(secretB), body.Code) {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid code")
				return
			}
			plain, hashes := generateRecoveryCodes()
			u.RecoveryHashes = hashes
			if err := users.UpsertUser(u); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, "persist error")
				return
			}
			writeJSON(w, map[string]any{"ok": true, "recovery_codes": plain})
		})

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
			list, _ := pools.ListPools(r.Context())
			writeJSON(w, list)
		})

		// Pools: allowed roots for shares (mounted pool paths)
		pr.Get("/api/pools/roots", func(w http.ResponseWriter, r *http.Request) {
			roots, err := poolroots.AllowedRoots()
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, map[string]any{"roots": roots})
		})

		pr.With(adminRequired).Post("/api/pools/plan-create", func(w http.ResponseWriter, r *http.Request) {
			var req pools.PlanRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if err := pools.EnsureDevicesFree(r.Context(), req.Devices); err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			client := handlers.AgentClientFactory()
			var planResp map[string]any
			_ = client.PostJSON(r.Context(), "/v1/btrfs/create", map[string]any{
				"devices": req.Devices,
				"raid":    req.Raid,
				"label":   req.Label,
				"dry_run": true,
			}, &planResp)
			writeJSON(w, planResp)
		})

		pr.With(adminRequired).Post("/api/pools/create", func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Confirm") != "yes" {
				httpx.WriteError(w, http.StatusPreconditionRequired, "confirm header required")
				return
			}
			var req pools.PlanRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if err := pools.EnsureDevicesFree(r.Context(), req.Devices); err != nil {
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			client := agentclient.New("/run/nos-agent.sock")
			var resp map[string]any
			err := client.PostJSON(r.Context(), "/v1/btrfs/create", map[string]any{
				"devices": req.Devices,
				"raid":    req.Raid,
				"label":   req.Label,
				"dry_run": false,
			}, &resp)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, resp)
		})

		// Pools: candidates for import
		pr.With(adminRequired).Get("/api/pools/candidates", func(w http.ResponseWriter, r *http.Request) {
			list, err := pools.ListPools(r.Context())
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, list)
		})

		// Pools: import (mount) by device or UUID
		pr.With(adminRequired).Post("/api/pools/import", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				DeviceOrUUID string `json:"device_or_uuid"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.TrimSpace(body.DeviceOrUUID) == "" {
				httpx.WriteError(w, http.StatusBadRequest, "device_or_uuid required")
				return
			}
			target := filepath.Join("/mnt", strings.ReplaceAll(body.DeviceOrUUID, "/", "_"))
			client := agentclient.New("/run/nos-agent.sock")
			var resp map[string]any
			if err := client.PostJSON(r.Context(), "/v1/btrfs/mount", map[string]any{"uuid_or_device": body.DeviceOrUUID, "target": target}, &resp); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, map[string]any{"ok": true, "mount": target})
		})

		// Shares
		pr.Get("/api/shares", func(w http.ResponseWriter, r *http.Request) {
			st := shares.NewStore(cfg.SharesPath)
			writeJSON(w, st.List())
		})
		// SMB users proxy
		pr.Get("/api/smb/users", func(w http.ResponseWriter, r *http.Request) {
			client := agentclient.New("/run/nos-agent.sock")
			var out struct {
				Users []string `json:"users"`
			}
			if err := client.GetJSON(r.Context(), "/v1/smb/users", &out); err != nil {
				// Graceful fallback
				writeJSON(w, []string{})
				return
			}
			writeJSON(w, out.Users)
		})
		pr.With(adminRequired).Post("/api/shares", handlers.HandleCreateShare(cfg))

		pr.With(adminRequired).Post("/api/smb/users", func(w http.ResponseWriter, r *http.Request) {
			var body struct{ Username, Password string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			client := agentclient.New("/run/nos-agent.sock")
			var resp map[string]any
			if err := client.PostJSON(r.Context(), "/v1/smb/user-create", map[string]any{"username": body.Username, "password": body.Password}, &resp); err != nil {
				// If agent returned HTTPError 400, propagate 400
				if he, ok := err.(*agentclient.HTTPError); ok && he.Status == http.StatusBadRequest {
					httpx.WriteError(w, http.StatusBadRequest, he.Body)
					return
				}
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, map[string]any{"ok": true})
		})
		pr.With(adminRequired).Delete("/api/shares/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			st := shares.NewStore(cfg.SharesPath)
			sh, ok := st.GetByID(id)
			if ok {
				// Best-effort: in dev/test on Windows or when the agent socket isn't present, skip agent calls
				var client *agentclient.Client
				if runtime.GOOS != "windows" {
					if _, err := os.Stat("/run/nos-agent.sock"); err == nil {
						client = agentclient.New("/run/nos-agent.sock")
					}
				}
				if sh.Type == "smb" {
					path := filepath.Join(cfg.EtcDir, "samba", "smb.conf.d", "nos-"+sh.Name+".conf")
					if client != nil {
						_ = client.PostJSON(r.Context(), "/v1/fs/write", map[string]any{"path": path, "content": "", "mode": "0644", "owner": "root", "group": "root"}, nil)
					}
				}
				if sh.Type == "nfs" {
					path := filepath.Join(cfg.EtcDir, "exports.d", "nos-"+sh.Name+".exports")
					if client != nil {
						_ = client.PostJSON(r.Context(), "/v1/fs/write", map[string]any{"path": path, "content": ""}, nil)
					}
				}
				_ = st.Delete(id)
				if client != nil {
					_ = client.PostJSON(r.Context(), "/v1/service/reload", map[string]any{"name": "smb"}, nil)
					_ = client.PostJSON(r.Context(), "/v1/service/reload", map[string]any{"name": "nfs"}, nil)
				}
			}
			w.WriteHeader(http.StatusNoContent)
		})

		pr.Get("/api/apps", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, apps.Catalog(cfg.AppsInstallDir))
		})

		pr.Get("/api/apps/{id}/status", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			for _, a := range apps.Catalog(cfg.AppsInstallDir) {
				if a.ID == id {
					writeJSON(w, a)
					return
				}
			}
			httpx.WriteError(w, http.StatusNotFound, "not found")
		})

		pr.With(adminRequired).Post("/api/apps/install", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				ID     string
				Config map[string]any
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ID == "" {
				httpx.WriteError(w, http.StatusBadRequest, "id required")
				return
			}
			dir := filepath.Join(cfg.AppsInstallDir, body.ID)
			_ = os.MkdirAll(dir, 0o755)
			compose := apps.ComposeTemplate(body.ID)
			if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0o644); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			unit := apps.UnitTemplate(body.ID, dir)
			client := agentclient.New("/run/nos-agent.sock")
			_ = client.PostJSON(r.Context(), "/v1/systemd/install-app", map[string]any{"id": body.ID, "unit_text": unit}, nil)
			_ = client.PostJSON(r.Context(), "/v1/app/compose-up", map[string]any{"id": body.ID, "dir": dir}, nil)
			writeJSON(w, map[string]any{"ok": true})
		})

		pr.With(adminRequired).Post("/api/apps/uninstall", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				ID    string
				Force bool
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ID == "" {
				httpx.WriteError(w, http.StatusBadRequest, "id required")
				return
			}
			dir := filepath.Join(cfg.AppsInstallDir, body.ID)
			client := agentclient.New("/run/nos-agent.sock")
			_ = client.PostJSON(r.Context(), "/v1/app/compose-down", map[string]any{"id": body.ID, "dir": dir}, nil)
			_ = client.PostJSON(r.Context(), "/v1/systemd/disable-app", map[string]any{"id": body.ID}, nil)
			_ = os.Remove(filepath.Join(dir, "docker-compose.yml"))
			_ = os.Remove(dir)
			writeJSON(w, map[string]any{"ok": true})
		})

		pr.Get("/api/remote/status", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"mode": "lan-only", "https": true})
		})

		// Support bundle
		pr.Get("/api/support/bundle", handleSupportBundle(cfg))

		// Firewall
		pr.Get("/api/firewall/status", func(w http.ResponseWriter, r *http.Request) {
			st, _ := firewall.Detect()
			mode, _ := firewall.ReadMode(filepath.Join(cfg.EtcDir, "nos", "firewall", "mode"))
			if mode == "" {
				mode = "lan-only"
			}
			st.Mode = mode
			writeJSON(w, st)
		})
		pr.With(adminRequired).Post("/api/firewall/plan", func(w http.ResponseWriter, r *http.Request) {
			var body struct{ Mode string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			rules := firewall.BuildRules(body.Mode)
			writeJSON(w, map[string]any{"rules": rules})
		})
		pr.With(adminRequired).Post("/api/firewall/apply", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Mode           string
				TwoFactorToken string
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.ToLower(body.Mode) != "lan-only" {
				if s, ok := codec.DecodeFromRequest(r); !ok || !s.TwoFA {
					httpx.WriteError(w, http.StatusForbidden, "2FA required")
					return
				}
			}
			st, _ := firewall.Detect()
			if st.UFWPresent || st.FirewalldPresent {
				httpx.WriteError(w, http.StatusConflict, "UFW or firewalld active")
				return
			}
			rules := firewall.BuildRules(body.Mode)
			client := agentclient.New("/run/nos-agent.sock")
			var resp map[string]any
			if err := client.PostJSON(r.Context(), "/v1/firewall/apply", map[string]any{"ruleset_text": rules, "persist": true}, &resp); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			_ = firewall.WriteMode(filepath.Join(cfg.EtcDir, "nos", "firewall", "mode"), body.Mode)
			writeJSON(w, map[string]any{"ok": true})
		})

		// Snapshots
		pr.Get("/api/pools/{id}/snapshots", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			list, _ := pools.ListSnapshots(r.Context(), id)
			writeJSON(w, list)
		})

		// Updates: check
		pr.Get("/api/updates/check", func(w http.ResponseWriter, r *http.Request) {
			client := handlers.AgentClientFactory()
			var planResp map[string]any
			_ = client.PostJSON(r.Context(), "/v1/updates/plan", map[string]any{}, &planResp)
			// attach snapshot targets (best-effort)
			roots, _ := poolroots.AllowedRoots()
			writeJSON(w, map[string]any{"plan": planResp, "snapshot_roots": roots})
		})

		// Updates: apply
		pr.With(adminRequired).Post("/api/updates/apply", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Packages []string `json:"packages"`
				Snapshot bool     `json:"snapshot"`
				Confirm  string   `json:"confirm"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.ToLower(body.Confirm) != "yes" {
				httpx.WriteError(w, http.StatusPreconditionRequired, "confirm\u003dyes required")
				return
			}
			client := handlers.AgentClientFactory()
			// create tx and persist initial state
			txID := generateUUID()
			tx := snapdb.UpdateTx{TxID: txID, StartedAt: time.Now().UTC(), Packages: body.Packages, Reason: "pre-update"}
			_ = snapdb.Append(tx)
			// load snapshot targets via pools roots (simplified: allowed roots)
			roots, _ := poolroots.AllowedRoots()
			if body.Snapshot {
				for _, p := range roots {
					var sresp struct {
						OK                 bool `json:"ok"`
						ID, Type, Location string
					}
					if err := client.PostJSON(r.Context(), "/v1/snapshot/create", map[string]any{"path": p, "mode": "auto", "reason": "pre-update"}, &sresp); err != nil {
						mark := false
						now := time.Now().UTC()
						tx.FinishedAt = &now
						tx.Success = &mark
						tx.Notes = "snapshot failed: " + errString(err)
						_ = snapdb.Append(tx)
						httpx.WriteError(w, http.StatusInternalServerError, "snapshot failed")
						return
					}
					// append target on success
					tx.Targets = append(tx.Targets, snapdb.SnapshotTarget{ID: sresp.ID, Path: p, Type: sresp.Type, Location: sresp.Location, CreatedAt: time.Now().UTC()})
				}
			}
			// perform updates apply on agent
			var applyResp map[string]any
			if err := client.PostJSON(r.Context(), "/v1/updates/apply", map[string]any{"packages": body.Packages}, &applyResp); err != nil {
				mark := false
				now := time.Now().UTC()
				tx.FinishedAt = &now
				tx.Success = &mark
				tx.Notes = "apply failed: " + errString(err)
				_ = snapdb.Append(tx)
				httpx.WriteError(w, http.StatusInternalServerError, "updates apply failed")
				return
			}
			// success
			mark := true
			now := time.Now().UTC()
			tx.FinishedAt = &now
			tx.Success = &mark
			_ = snapdb.Append(tx)
			writeJSON(w, map[string]any{"ok": true, "tx_id": txID, "snapshots_count": len(tx.Targets), "updates_count": len(applyResp)})
		})

		// Snapshots: prune
		pr.With(adminRequired).Post("/api/snapshots/prune", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				KeepPerTarget int `json:"keep_per_target"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.KeepPerTarget <= 0 {
				body.KeepPerTarget = 5
			}
			client := handlers.AgentClientFactory()
			var resp map[string]any
			if err := client.PostJSON(r.Context(), "/v1/snapshot/prune", map[string]any{"keep_per_target": body.KeepPerTarget}, &resp); err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, resp)
		})

		// Updates: rollback
		pr.With(adminRequired).Post("/api/updates/rollback", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				TxID    string `json:"tx_id"`
				Confirm string `json:"confirm"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.ToLower(body.Confirm) != "yes" {
				httpx.WriteError(w, http.StatusPreconditionRequired, "confirm\u003dyes required")
				return
			}
			orig, err := snapdb.FindByTx(body.TxID)
			if err != nil {
				httpx.WriteError(w, http.StatusNotFound, "tx not found")
				return
			}
			client := handlers.AgentClientFactory()
			// start rollback tx record
			roll := snapdb.UpdateTx{TxID: generateUUID(), StartedAt: time.Now().UTC(), Packages: orig.Packages, Reason: "rollback"}
			for _, t := range orig.Targets {
				var resp map[string]any
				if err := client.PostJSON(r.Context(), "/v1/snapshot/rollback", map[string]any{
					"path": t.Path, "snapshot_id": t.ID, "type": t.Type,
				}, &resp); err != nil {
					mark := false
					now := time.Now().UTC()
					roll.FinishedAt = &now
					roll.Success = &mark
					roll.Notes = "rollback failed for target " + t.Path + ": " + err.Error()
					_ = snapdb.Append(roll)
					httpx.WriteError(w, http.StatusInternalServerError, "rollback failed")
					return
				}
			}
			// persist rollback record and adjust ordering by writing a small "touch"
			now := time.Now().UTC()
			roll.FinishedAt = &now
			okMark := true
			roll.Success = &okMark
			roll.Notes = "rollback of " + orig.TxID
			_ = snapdb.Append(roll)
			writeJSON(w, map[string]any{"ok": true})
		})

		// Snapshots DB: recent
		pr.Get("/api/snapshots/recent", func(w http.ResponseWriter, r *http.Request) {
			list, err := snapdb.ListRecent(20)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			// project limited fields
			out := make([]map[string]any, 0, len(list))
			for _, tx := range list {
				miniTargets := make([]map[string]any, 0, len(tx.Targets))
				for _, t := range tx.Targets {
					miniTargets = append(miniTargets, map[string]any{
						"id": t.ID, "type": t.Type, "location": t.Location,
					})
				}
				ok := false
				if tx.Success != nil {
					ok = *tx.Success
				}
				out = append(out, map[string]any{
					"tx_id":    tx.TxID,
					"time":     tx.StartedAt,
					"packages": tx.Packages,
					"targets":  miniTargets,
					"success":  ok,
				})
			}
			writeJSON(w, out)
		})

		// Snapshots DB: by tx id
		pr.Get("/api/snapshots/{tx_id}", func(w http.ResponseWriter, r *http.Request) {
			txID := chi.URLParam(r, "tx_id")
			tx, err := snapdb.FindByTx(txID)
			if err != nil {
				httpx.WriteError(w, http.StatusNotFound, "tx not found")
				return
			}
			writeJSON(w, tx)
		})

		pr.With(adminRequired).Post("/api/pools/{id}/snapshots", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			var body struct {
				Subvol string
				Name   string
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			client := agentclient.New("/run/nos-agent.sock")
			var resp map[string]any
			err := client.PostJSON(r.Context(), "/v1/btrfs/snapshot", map[string]any{"path": body.Subvol, "name": body.Name}, &resp)
			if err != nil {
				httpx.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}
			_ = id // unused for now
			writeJSON(w, resp)
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

func generateUUID() string {
	const hex = "0123456789abcdef"
	var b [16]byte
	n := time.Now().UnixNano()
	for i := 0; i < 16; i++ {
		b[i] = byte(n >> (i * 8))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	out := make([]byte, 36)
	idx := 0
	for i := 0; i < 16; i++ {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			out[idx] = '-'
			idx++
		}
		out[idx] = hex[b[i]>>4]
		idx++
		out[idx] = hex[b[i]&0x0f]
		idx++
	}
	return string(out)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// naive in-memory rate limiter
type rateLimiter struct {
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{hits: make(map[string][]time.Time), max: max, window: window}
}

func (r *rateLimiter) Allow(key string) bool {
	now := time.Now()
	lst := r.hits[key]
	cut := now.Add(-1 * r.window)
	kept := make([]time.Time, 0, len(lst))
	for _, t := range lst {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= r.max {
		r.hits[key] = kept
		return false
	}
	kept = append(kept, now)
	r.hits[key] = kept
	return true
}

// setup token helpers using secret.key
func setupEncodeToken(cfg config.Config, payload map[string]any) (string, error) {
	key, err := os.ReadFile(cfg.SecretPath)
	if err != nil {
		return "", err
	}
	sc := securecookie.New(key, nil)
	sc.MaxAge(600)
	return sc.Encode("nos_setup", payload)
}

func setupDecodeToken(cfg config.Config, tok string) (map[string]any, error) {
	key, err := os.ReadFile(cfg.SecretPath)
	if err != nil {
		return nil, err
	}
	sc := securecookie.New(key, nil)
	var out map[string]any
	if err := sc.Decode("nos_setup", tok, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// simple validators used during setup
func validUsername(s string) bool {
	if len(s) < 3 || len(s) > 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func validPassword(p string) bool {
	if len(p) < 12 {
		return false
	}
	hasLower, hasUpper, hasDigit, hasOther := false, false, false, false
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasOther = true
		}
	}
	count := 0
	if hasLower {
		count++
	}
	if hasUpper {
		count++
	}
	if hasDigit {
		count++
	}
	if hasOther {
		count++
	}
	return count >= 3
}

func mustJSON(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
