package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"nithronos/backend/nosd/handlers"
	"nithronos/backend/nosd/internal/apps"
	pwhash "nithronos/backend/nosd/internal/auth/hash"
	"nithronos/backend/nosd/internal/auth/session"
	userstore "nithronos/backend/nosd/internal/auth/store"
	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/disks"
	"nithronos/backend/nosd/internal/pools"
	"nithronos/backend/nosd/internal/ratelimit"
	"nithronos/backend/nosd/internal/sessions"
	"nithronos/backend/nosd/internal/shares"
	"nithronos/backend/nosd/pkg/agentclient"
	"nithronos/backend/nosd/pkg/auth"
	"nithronos/backend/nosd/pkg/firewall"
	"nithronos/backend/nosd/pkg/httpx"
	poolroots "nithronos/backend/nosd/pkg/pools"
	"nithronos/backend/nosd/pkg/snapdb"

	"nithronos/backend/nosd/internal/fsatomic"

	"strconv"

	firstboot "nithronos/backend/nosd/internal/setup/firstboot"

	"github.com/gorilla/securecookie"
)

// agentMetricsClient implements AgentMetricsClient to fetch text metrics from nos-agent
type agentMetricsClient struct{ socket string }

func (a agentMetricsClient) FetchMetrics(ctx context.Context) ([]byte, error) {
	cli := agentclient.New(a.socket)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix/metrics", nil)
	res, err := cli.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func Logger(cfg config.Config) *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	level := currentLevel
	logger := log.Logger.Level(level).With().Timestamp().Logger()
	return &logger
}

func NewRouter(cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(zerologMiddleware(Logger(cfg), cfg))
	r.Use(securityHeaders)

	// Dynamic CORS based on runtime config
	SetRuntimeCORSOrigin(cfg.CORSOrigin)
	r.Use(DynamicCORS)

	// Observability endpoints: metrics and pprof
	if cfg.MetricsEnabled {
		r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
			// very simple allowlist by exact ip match or prefix
			if len(cfg.MetricsAllowlist) > 0 {
				ip := clientIP(r, cfg)
				allowed := false
				for _, a := range cfg.MetricsAllowlist {
					if a == ip || (strings.HasSuffix(a, ".") && strings.HasPrefix(ip, a)) {
						allowed = true
						break
					}
				}
				if !allowed {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			var b strings.Builder
			b.WriteString("nosd_up 1\n")
			// pool metrics (best-effort)
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if list, err := pools.ListPools(ctx); err == nil {
				var total uint64
				var used uint64
				for _, p := range list {
					total += p.Size
					used += p.Used
				}
				b.WriteString(fmt.Sprintf("pool_total_bytes %d\n", total))
				b.WriteString(fmt.Sprintf("pool_used_bytes %d\n", used))
			}
			// SMART metrics for common devices (best-effort)
			for _, dev := range []string{"/dev/sda", "/dev/nvme0n1"} {
				client := agentclient.New("/run/nos-agent.sock")
				var out map[string]any
				if err := client.GetJSON(r.Context(), "/v1/smart?device="+dev, &out); err == nil {
					if t, ok := out["temperature_c"].(float64); ok {
						b.WriteString(fmt.Sprintf("smart_disk_temp_celsius{dev=\"%s\"} %g\n", dev, t))
					}
					if st, ok := out["passed"].(bool); ok {
						if st {
							b.WriteString(fmt.Sprintf("smart_pass{dev=\"%s\"} 1\n", dev))
						} else {
							b.WriteString(fmt.Sprintf("smart_pass{dev=\"%s\"} 0\n", dev))
						}
					}
				}
			}
			// Btrfs tx progress (best-effort gauges set by executor)
			if p := currentBalancePercent(); p >= 0 {
				b.WriteString(fmt.Sprintf("btrfs_balance_percent %g\n", p))
			}
			if p := currentReplacePercent(); p >= 0 {
				b.WriteString(fmt.Sprintf("btrfs_replace_percent %g\n", p))
			}
			_, _ = w.Write([]byte(b.String()))
		})
		// Combined metrics endpoint: nosd + agent
		r.Get("/metrics/all", func(w http.ResponseWriter, r *http.Request) {
			NewCombinedMetricsHandler(prom.DefaultGatherer, agentMetricsClient{socket: cfg.AgentSocket()}).ServeHTTP(w, r)
		})
	}

	if cfg.PprofEnabled {
		// Guard pprof: localhost only
		r.Mount("/debug/pprof", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			if i := strings.LastIndex(ip, ":"); i >= 0 {
				ip = ip[:i]
			}
			if ip != "127.0.0.1" && ip != "::1" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			http.DefaultServeMux.ServeHTTP(w, r)
		}))
	}

	// Init stores
	store, _ := auth.NewStore(cfg.UsersPath)
	users, _ := userstore.New(cfg.UsersPath)
	codec := auth.NewSessionCodec(cfg.SessionHashKey, cfg.SessionBlockKey)
	// Disk-backed session and ratelimit stores
	sessStore := sessions.New(filepath.Join("/var/lib/nos", "sessions.json"))
	rlStore := ratelimit.New(filepath.Join("/var/lib/nos", "ratelimit.json"))
	mgr := session.New(filepath.Join("/var/lib/nos", "sessions.json"))

	// On startup: if first boot and OTP exists/valid, log it
	func() {
		// Determine if setup complete by checking users on disk (fresh load)
		us, _ := userstore.New(cfg.UsersPath)
		if us == nil || !us.HasAdmin() {
			// Load first-boot OTP state
			var st struct {
				OTP       string `json:"otp"`
				CreatedAt string `json:"created_at"`
				Used      bool   `json:"used"`
			}
			if b, err := os.ReadFile(cfg.FirstBootPath); err == nil {
				_ = json.Unmarshal(b, &st)
				if st.OTP != "" && !st.Used {
					if t, err := time.Parse(time.RFC3339, st.CreatedAt); err == nil && time.Since(t) < 15*time.Minute {
						Logger(cfg).Info().Msgf("First-boot OTP: %s (valid 15m)", st.OTP)
					}
				}
			}
		}
	}()

	// Session verification middleware for server-side binding (non-enforcing)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, sid, ok := decodeSessionParts(r, cfg)
			if !ok || uid == "" || sid == "" {
				next.ServeHTTP(w, r)
				return
			}
			ua := r.Header.Get("User-Agent")
			ip := clientIP(r, cfg)
			if id, ok2 := mgr.Verify(sid, ua, ip); !ok2 || id != uid {
				// do not enforce; continue without rejecting
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// SessionContext middleware (parse cookies only; no auth enforcement)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// parse nos_session; ignore failures
			uid, sid, ok := decodeSessionParts(r, cfg)
			if ok {
				// attach to context via headers (lightweight)
				r.Header.Set("X-UID", uid)
				r.Header.Set("X-SID", sid)
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "version": "0.1.0"})
	})

	// Storage: block device inventory
	r.Get("/api/v1/storage/devices", handleListDevices)
	// SMART health proxy
	r.Get("/api/v1/health/smart", handleSmartProxy)

	// Storage: block device inventory
	r.Get("/api/v1/storage/devices", handleListDevices)

	// Recovery routes (localhost only)
	if cfg.RecoveryMode {
		r.Route("/api/v1/recovery", func(rr chi.Router) {
			rr.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ip := r.RemoteAddr
					if i := strings.LastIndex(ip, ":"); i >= 0 {
						ip = ip[:i]
					}
					if ip != "127.0.0.1" && ip != "::1" {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					next.ServeHTTP(w, r)
				})
			})
			rr.Post("/reset-password", func(w http.ResponseWriter, r *http.Request) {
				var body struct{ Username, Password string }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.TrimSpace(body.Username) == "" || strings.TrimSpace(body.Password) == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				users, err := userstore.New(cfg.UsersPath)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := users.FindByUsername(strings.ToLower(body.Username))
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				h, herr := pwhash.HashPassword(body.Password)
				if herr != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u.PasswordHash = h
				u.LockedUntil = ""
				u.FailedAttempts = 0
				_ = users.UpsertUser(u)
				writeJSON(w, map[string]any{"ok": true})
			})
			rr.Post("/disable-2fa", func(w http.ResponseWriter, r *http.Request) {
				var body struct{ Username string }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if strings.TrimSpace(body.Username) == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				users, err := userstore.New(cfg.UsersPath)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				u, err := users.FindByUsername(strings.ToLower(body.Username))
				if err != nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				u.TOTPEnc = ""
				u.RecoveryHashes = nil
				_ = users.UpsertUser(u)
				writeJSON(w, map[string]any{"ok": true})
			})
			rr.Post("/generate-otp", func(w http.ResponseWriter, r *http.Request) {
				// Regenerate a one-time setup OTP (best-effort)
				var st struct {
					OTP       string `json:"otp"`
					CreatedAt string `json:"created_at"`
					Used      bool   `json:"used"`
				}
				st.OTP = genOTP6()
				st.CreatedAt = time.Now().UTC().Format(time.RFC3339)
				st.Used = false
				_ = os.MkdirAll(filepath.Dir(cfg.FirstBootPath), 0o755)
				_ = fsatomic.SaveJSON(r.Context(), cfg.FirstBootPath, st, 0o600)
				writeJSON(w, map[string]any{"otp": st.OTP})
			})
		})
	}

	// Agent registration (bootstrap trust)
	r.Post("/api/v1/agents/register", func(w http.ResponseWriter, r *http.Request) {
		if !cfg.AllowAgentRegistration {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var body struct {
			Token string `json:"token"`
			Node  string `json:"node"`
			Arch  string `json:"arch"`
			OS    string `json:"os"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		// compare against bootstrap token
		bootTok, _ := os.ReadFile("/etc/nos/agent-token")
		if len(bootTok) == 0 || strings.TrimSpace(body.Token) != strings.TrimSpace(string(bootTok)) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// rotate per-agent token and persist (very simple JSON list)
		agentsPath := filepath.Join("/var/lib/nos", "agents.json")
		type agentRec struct{ ID, Token, Node, Arch, OS, CreatedAt string }
		var list []agentRec
		if b, err := os.ReadFile(agentsPath); err == nil {
			_ = json.Unmarshal(b, &list)
		}
		id := generateUUID()
		tok := generateUUID()
		rec := agentRec{ID: id, Token: tok, Node: body.Node, Arch: body.Arch, OS: body.OS, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
		list = append(list, rec)
		_ = os.MkdirAll(filepath.Dir(agentsPath), 0o755)
		_ = fsatomic.SaveJSON(r.Context(), agentsPath, list, 0o600)
		writeJSON(w, map[string]any{"id": id, "token": tok})
	})

	// Setup routes are always registered, but gated with 410 when setup is complete
	r.Route("/api/setup", func(sr chi.Router) {
		sr.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Evaluate setup completion from disk on every request (robust against file changes)
				us, _ := userstore.New(cfg.UsersPath)
				if us != nil && us.HasAdmin() {
					httpx.WriteTypedError(w, http.StatusGone, "setup.complete", "Setup already completed", 0)
					return
				}
				next.ServeHTTP(w, r)
			})
		})
		sr.Get("/state", func(w http.ResponseWriter, r *http.Request) {
			// Compute firstBoot and whether an OTP is currently required (exists and valid)
			firstBoot := true
			if us, _ := userstore.New(cfg.UsersPath); us != nil && us.HasAdmin() {
				firstBoot = false
			}
			otpRequired := false
			if st, err := firstboot.New(cfg.FirstBootPath).Load(); err == nil && st != nil {
				if time.Now().Before(st.ExpiresAt) && st.OTP != "" {
					otpRequired = true
				}
			}
			writeJSON(w, map[string]any{"firstBoot": firstBoot, "otpRequired": otpRequired})
		})

		// Rate limiter (persisted): per-IP cfg.RateOTPPerMin per minute for setup endpoints
		sr.Post("/verify-otp", func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, cfg)
			otpWin := time.Duration(cfg.RateOTPWindowSec) * time.Second
			if otpWin <= 0 {
				otpWin = time.Minute
			}
			ok1, rem1, reset1 := rlStore.Allow("otp:ip:"+ip, cfg.RateOTPPerMin, otpWin)
			if !ok1 {
				retry := int(time.Until(reset1).Seconds())
				Logger(cfg).Warn().Str("event", "rate.limited").Str("route", "/api/setup/verify-otp").Str("key", "otp:ip:"+ip).Int("remaining", rem1).Int("retryAfterSec", retry).Msg("")
				httpx.WriteTypedError(w, http.StatusTooManyRequests, "rate.limited", "Too many attempts. Try later.", retry)
				return
			}
			var body struct{ OTP string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.OTP) != 6 {
				httpx.WriteTypedError(w, http.StatusBadRequest, "setup.otp.invalid", "Enter the 6-digit code", 0)
				return
			}
			st, err := firstboot.New(cfg.FirstBootPath).Load()
			if err != nil {
				if os.IsPermission(err) {
					httpx.WriteTypedError(w, http.StatusInternalServerError, "storage_error", "setup storage not writable", 0)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if st == nil || st.OTP == "" || st.OTP != body.OTP {
				httpx.WriteTypedError(w, http.StatusBadRequest, "setup.otp.invalid", "Invalid one-time code", 0)
				return
			}
			if time.Now().After(st.ExpiresAt) {
				httpx.WriteTypedError(w, http.StatusGone, "setup.otp.expired", "Your code expired. Request a new one.", 0)
				return
			}
			payload := map[string]any{"purpose": "setup", "exp": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)}
			val, err := setupEncodeToken(cfg, payload)
			if err != nil {
				Logger(cfg).Error().Str("event", "setup.token.error").Err(err).Msg("")
				httpx.WriteTypedError(w, http.StatusInternalServerError, "secret_unreadable", "secret.key not readable", 0)
				return
			}
			writeJSON(w, map[string]any{"ok": true, "token": val})
		})

		sr.Post("/create-admin", func(w http.ResponseWriter, r *http.Request) {
			if users == nil {
				httpx.WriteTypedError(w, http.StatusInternalServerError, "store.lock", "User store unavailable", 0)
				return
			}
			authz := r.Header.Get("Authorization")
			const prefix = "Bearer "
			if !strings.HasPrefix(authz, prefix) {
				httpx.WriteTypedError(w, http.StatusUnauthorized, "setup.session.invalid", "Missing setup bearer token", 0)
				return
			}
			tok := strings.TrimSpace(authz[len(prefix):])
			claims, err := setupDecodeToken(cfg, tok)
			if err != nil {
				httpx.WriteTypedError(w, http.StatusUnauthorized, "setup.session.invalid", "Invalid setup token", 0)
				return
			}
			if claims["purpose"] != "setup" {
				httpx.WriteTypedError(w, http.StatusUnauthorized, "setup.session.invalid", "Invalid setup token", 0)
				return
			}
			if expStr, ok := claims["exp"].(string); ok {
				if t, err := time.Parse(time.RFC3339, expStr); err != nil || time.Now().After(t) {
					httpx.WriteTypedError(w, http.StatusBadRequest, "setup.otp.expired", "Setup token expired", 0)
					return
				}
			} else {
				httpx.WriteTypedError(w, http.StatusUnauthorized, "setup.session.invalid", "Invalid setup token", 0)
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
				httpx.WriteTypedError(w, http.StatusBadRequest, "input.invalid", "Invalid username", 0)
				return
			}
			if !validPassword(body.Password) {
				httpx.WriteTypedError(w, http.StatusBadRequest, "input.weak_password", "Choose a stronger password", 0)
				return
			}
			if _, err := users.FindByUsername(uname); err == nil {
				httpx.WriteTypedError(w, http.StatusConflict, "input.username_taken", "Username is taken", 0)
				return
			}
			phc, err := pwhash.HashPassword(body.Password)
			if err != nil {
				Logger(cfg).Error().Str("event", "setup.hash.error").Err(err).Msg("")
				httpx.WriteTypedError(w, http.StatusInternalServerError, "store.atomic_fail", "Internal error", 0)
				return
			}
			now := time.Now().UTC().Format(time.RFC3339)
			u := userstore.User{ID: generateUUID(), Username: uname, PasswordHash: phc, Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now}
			if body.EnableTOTP {
				u.TOTPEnc = "pending"
			}
			if err := users.UpsertUser(u); err != nil {
				code := "store.atomic_fail"
				if os.IsPermission(err) || os.IsNotExist(err) || strings.Contains(strings.ToLower(err.Error()), "permission denied") {
					code = "setup.write_failed"
					// Emit clear hint for permission issues
					info := dirPermInfo(filepath.Dir(cfg.UsersPath))
					Logger(cfg).Error().Str("event", "setup.persist.error").Str("code", code).Str("hint", info).Err(err).Msg("")
				} else {
					Logger(cfg).Error().Str("event", "setup.persist.error").Str("code", code).Err(err).Msg("")
				}
				httpx.WriteErrorWithDetails(w, http.StatusInternalServerError, code, "Service cannot write /etc/nos/users.json", map[string]any{"path": cfg.UsersPath})
				return
			}
			// Success: remove first-boot state so OTP stops printing on restarts (best-effort)
			_ = os.Remove(cfg.FirstBootPath)
			w.WriteHeader(http.StatusNoContent)
		})
	})

	// Recovery: local-only endpoint to clear first-boot state and optionally users
	r.Post("/api/setup/recover", func(w http.ResponseWriter, r *http.Request) {
		// Guard: localhost only
		ip := r.RemoteAddr
		if i := strings.LastIndex(ip, ":"); i >= 0 {
			ip = ip[:i]
		}
		if ip != "127.0.0.1" && ip != "::1" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		var body struct {
			Confirm     string `json:"confirm"`
			DeleteUsers bool   `json:"delete_users"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.ToLower(strings.TrimSpace(body.Confirm)) != "yes" {
			httpx.WriteTypedError(w, http.StatusPreconditionRequired, "confirm.required", "confirm=yes required", 0)
			return
		}
		// Best-effort deletes
		_ = os.Remove(cfg.FirstBootPath)
		if body.DeleteUsers {
			_ = os.Remove(cfg.UsersPath)
		}
		writeJSON(w, map[string]any{"ok": true})
	})

	// Remove legacy login limiter seed (persisted store is the single source of truth)
	// (intentionally left blank)

	// First admin creation (consumes first-boot OTP)
	r.Post("/api/setup/first-admin", func(w http.ResponseWriter, r *http.Request) {
		if users == nil {
			httpx.WriteError(w, http.StatusInternalServerError, "user store unavailable")
			return
		}
		if users.HasAdmin() {
			httpx.WriteTypedError(w, http.StatusGone, "setup.complete", "Setup already completed", 0)
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
		_ = fsatomic.SaveJSON(context.TODO(), cfg.FirstBootPath, st, 0o600)
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
		ip := clientIP(r, cfg)
		loginWin := time.Duration(cfg.RateLoginWindowSec) * time.Second
		if loginWin <= 0 {
			loginWin = 15 * time.Minute
		}
		okIP, _, resetIP := rlStore.Allow("login:ip:"+ip, cfg.RateLoginPer15m, loginWin)
		okUser, _, resetUser := rlStore.Allow("login:user:"+strings.ToLower(uname), cfg.RateLoginPer15m, loginWin)
		if !okIP || !okUser {
			retry := resetIP
			if time.Until(resetUser) > 0 && resetUser.After(retry) {
				retry = resetUser
			}
			Logger(cfg).Warn().Str("event", "rate.limited").Str("key", "login").Str("ip", ip).Int("limit", cfg.RateLoginPer15m).Time("resetAt", retry).Msg("")
			w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(retry).Seconds())))
			httpx.WriteError(w, http.StatusTooManyRequests, `{"error":{"code":"rate.limited","retryAfterSec":`+strconv.Itoa(int(time.Until(retry).Seconds()))+`}}`)
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
		// success: reset counters
		u.FailedAttempts = 0
		u.LockedUntil = ""
		_ = users.UpsertUser(u)
		if err := issueSessionCookies(w, cfg, u.ID, body.RememberMe); err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, "session error")
			return
		}
		// persist session record (best-effort)
		_ = sessStore.Upsert(sessions.Session{ID: generateUUID(), UserID: u.ID, Roles: u.Roles, ExpiresAt: time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)})
		// bind server-side session
		ua := r.Header.Get("User-Agent")
		ip = clientIP(r, cfg)
		rec, _ := mgr.Create(u.ID, ua, ip, 15*time.Minute)
		_ = issueSessionCookiesSID(w, cfg, u.ID, rec.SID, body.RememberMe)
		issueCSRFCookie(w)
		writeJSON(w, map[string]any{"ok": true})
	})

	// Record refresh events in sessions store (best-effort)
	r.Post("/api/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if uid, ok := decodeRefreshUID(r, cfg); ok {
			// rotate refresh; revoke all if reuse detected
			old := r.Header.Get("X-Refresh-ID")
			newID, reuse, _ := mgr.RotateRefresh(uid, strings.TrimSpace(old))
			if reuse {
				_ = mgr.RevokeAll(uid)
				clearAuthCookies(w)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = sessStore.Upsert(sessions.Session{ID: generateUUID(), UserID: uid, Roles: []string{"refresh"}, ExpiresAt: time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)})
			if err := issueSessionCookies(w, cfg, uid, true); err == nil {
				w.Header().Set("X-Refresh-ID", newID)
				writeJSON(w, map[string]any{"ok": true})
				return
			}
		}
		w.WriteHeader(http.StatusUnauthorized)
	})

	// Logout: clear cookies and remove persisted sessions for this user (best-effort)
	r.Post("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if uid, ok := decodeSessionUID(r, cfg); ok {
			_ = sessStore.DeleteByUserID(uid)
			_ = mgr.RevokeAll(uid)
		}
		clearAuthCookies(w)
		w.WriteHeader(http.StatusNoContent)
	})

	// Protected API group (auth required)
	r.Group(func(pr chi.Router) {
		pr.Use(func(next http.Handler) http.Handler { return requireAuth(next, codec, cfg) })
		// Session endpoints (self scope)
		pr.Get("/api/v1/auth/sessions", func(w http.ResponseWriter, r *http.Request) {
			uid, ok := decodeSessionUID(r, cfg)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			list := mgr.ListByUser(uid)
			// mark current
			curSID := r.Header.Get("X-SID")
			out := make([]map[string]any, 0, len(list))
			for _, s := range list {
				out = append(out, map[string]any{
					"sid":           s.SID,
					"createdAt":     s.CreatedAt,
					"lastSeenAt":    s.LastSeenAt,
					"ipPrefix":      s.IPHash,
					"uaFingerprint": s.UAHash,
					"current":       s.SID == curSID,
				})
			}
			writeJSON(w, out)
		})
		pr.Post("/api/v1/auth/sessions/revoke", func(w http.ResponseWriter, r *http.Request) {
			uid, ok := decodeSessionUID(r, cfg)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			var body struct{ Scope, SID string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Scope == "" {
				body.Scope = "current"
			}
			ip := r.RemoteAddr
			if h := r.Header.Get("X-Forwarded-For"); h != "" {
				ip = strings.Split(h, ",")[0]
			}
			switch body.Scope {
			case "current":
				cur := r.Header.Get("X-SID")
				if cur != "" {
					_ = mgr.RevokeSID(cur)
				}
				clearAuthCookies(w)
				Logger(cfg).Info().Str("event", "auth.session.revoke").Str("userId", uid).Str("scope", "current").Str("sid", cur).Str("ip", ip).Msg("")
			case "all":
				_ = mgr.RevokeAll(uid)
				clearAuthCookies(w)
				Logger(cfg).Info().Str("event", "auth.session.revoke").Str("userId", uid).Str("scope", "all").Str("ip", ip).Msg("")
			case "sid":
				if body.SID == "" {
					httpx.WriteError(w, http.StatusBadRequest, "sid required")
					return
				}
				// validate ownership
				owned := false
				for _, s := range mgr.ListByUser(uid) {
					if s.SID == body.SID {
						owned = true
						break
					}
				}
				if !owned {
					httpx.WriteError(w, http.StatusForbidden, "not your session")
					return
				}
				_ = mgr.RevokeSID(body.SID)
				Logger(cfg).Info().Str("event", "auth.session.revoke").Str("userId", uid).Str("scope", "sid").Str("sid", body.SID).Str("ip", ip).Msg("")
			default:
				httpx.WriteError(w, http.StatusBadRequest, "invalid scope")
				return
			}
			writeJSON(w, map[string]any{"ok": true})
		})
	})

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
		// Require auth via new opaque cookies or legacy session cookie (skip in tests when NOS_TEST_SKIP_AUTH=1)
		if os.Getenv("NOS_TEST_SKIP_AUTH") != "1" {
			pr.Use(func(next http.Handler) http.Handler { return requireAuth(next, codec, cfg) })
		}
		if os.Getenv("NOS_TEST_SKIP_AUTH") != "1" {
			pr.Use(requireCSRF)
		}

		// AdminRequired middleware: resolve current user and assert role
		adminRequired := func(next http.Handler) http.Handler {
			// Skip admin check in tests
			if os.Getenv("NOS_TEST_SKIP_AUTH") == "1" {
				return next
			}
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

		pr.With(adminRequired).Post("/api/v1/pools/plan-create", handlePlanCreateV1)

		// Health: alerts and manual SMART scan
		pr.Get("/api/v1/alerts", handleAlertsGet(cfg))
		pr.With(adminRequired).Post("/api/v1/health/scan", handleHealthScan(cfg))
		pr.With(adminRequired).Post("/api/v1/pools/apply-create", handleApplyCreate(cfg))
		pr.With(adminRequired).Get("/api/v1/pools/discover", handlePoolsDiscover)
		pr.With(adminRequired).Post("/api/v1/pools/import", handlePoolsImport(cfg))
		// Device operations (plan/apply)
		pr.With(adminRequired).Post("/api/v1/pools/{id}/plan-device", handlePlanDevice(cfg))
		pr.With(adminRequired).Post("/api/v1/pools/{id}/apply-device", handleApplyDevice(cfg))
		pr.With(adminRequired).Post("/api/v1/pools/{id}/plan-destroy", handlePlanDestroy(cfg))
		pr.With(adminRequired).Post("/api/v1/pools/{id}/apply-destroy", handleApplyDestroy(cfg))
		pr.With(adminRequired).Post("/api/v1/pools/scrub/start", handleScrubStart)
		pr.With(adminRequired).Get("/api/v1/pools/scrub/status", handleScrubStatus)
		pr.Get("/api/v1/pools/{id}", handlePoolDetail)
		pr.Get("/api/v1/pools/{id}/options", handlePoolOptionsGet(cfg))
		pr.With(adminRequired).Post("/api/v1/pools/{id}/options", handlePoolOptionsPost(cfg))

		pr.Get("/api/v1/schedules", handleSchedulesGet(cfg))
		pr.With(adminRequired).Post("/api/v1/schedules", handleSchedulesPost(cfg))
		pr.Get("/api/v1/pools/tx/{id}/status", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			var tx pools.Tx
			if ok, _ := fsatomic.LoadJSON(txPath(id), &tx); !ok {
				httpx.WriteError(w, http.StatusNotFound, "not found")
				return
			}
			writeJSON(w, tx)
		})
		pr.Get("/api/v1/pools/tx/{id}/log", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			cursorStr := r.URL.Query().Get("cursor")
			maxStr := r.URL.Query().Get("max")
			cursor, max := 0, 1000
			if i, err := strconv.Atoi(cursorStr); err == nil && i >= 0 {
				cursor = i
			}
			if i, err := strconv.Atoi(maxStr); err == nil && i > 0 && i <= 5000 {
				max = i
			}
			lines, next := readLogTail(id, cursor, max)
			writeJSON(w, map[string]any{"lines": lines, "nextCursor": next})
		})
		pr.Get("/api/v1/pools/tx/{id}/stream", handleTxStream)

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

// legacy in-memory rate limiter removed; persisted ratelimit store governs throttling

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

// clientIP extracts client IP. If TrustProxy is true, use last untrusted hop from X-Forwarded-For; otherwise use RemoteAddr.
func clientIP(r *http.Request, cfg config.Config) string {
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i >= 0 {
		ip = ip[:i]
	}
	if !(cfg.TrustProxy || RuntimeTrustProxy()) {
		return ip
	}
	h := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if h == "" {
		return ip
	}
	parts := strings.Split(h, ",")
	// take the last non-empty token
	for i := len(parts) - 1; i >= 0; i-- {
		p := strings.TrimSpace(parts[i])
		if p != "" {
			return p
		}
	}
	return ip
}

// genOTP6 generates a 6-digit OTP.
func genOTP6() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "000000"
	}
	n := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1000000
	return fmt.Sprintf("%06d", n)
}

func dirPermInfo(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	st := fi.Mode().Perm()
	// ls -ld-like summary (very simplified)
	ownerGid := ""
	if out, e := exec.Command("/usr/bin/stat", "-c", "%U:%G", path).CombinedOutput(); e == nil {
		ownerGid = strings.TrimSpace(string(out))
	}
	return fmt.Sprintf("ls -ld %s => mode %o owner %s; recommend: chown -R nos:nos %s && chmod 0750 %s", path, st, ownerGid, path, path)
}
