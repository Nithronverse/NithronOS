package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"nithronos/backend/nosd/internal/apps"
	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/disks"
	"nithronos/backend/nosd/internal/pools"
	"nithronos/backend/nosd/internal/shares"
	"nithronos/backend/nosd/pkg/agentclient"
	"nithronos/backend/nosd/pkg/auth"
	"nithronos/backend/nosd/pkg/firewall"
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
			list, _ := pools.ListPools(r.Context())
			writeJSON(w, list)
		})

		pr.Post("/api/pools/plan-create", func(w http.ResponseWriter, r *http.Request) {
			var req pools.PlanRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if err := pools.EnsureDevicesFree(r.Context(), req.Devices); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
				return
			}
			client := agentclient.New("/run/nos-agent.sock")
			var planResp map[string]any
			_ = client.PostJSON(r.Context(), "/v1/btrfs/create", map[string]any{
				"devices": req.Devices,
				"raid":    req.Raid,
				"label":   req.Label,
				"dry_run": true,
			}, &planResp)
			writeJSON(w, planResp)
		})

		pr.Post("/api/pools/create", func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Confirm") != "yes" {
				w.WriteHeader(http.StatusPreconditionRequired)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "confirm header required"})
				return
			}
			var req pools.PlanRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			if err := pools.EnsureDevicesFree(r.Context(), req.Devices); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
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
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, resp)
		})

		// Shares
		pr.Get("/api/shares", func(w http.ResponseWriter, r *http.Request) {
			st := shares.NewStore(cfg.SharesPath)
			writeJSON(w, st.List())
		})
		pr.Post("/api/shares", func(w http.ResponseWriter, r *http.Request) {
			var body shares.Share
			_ = json.NewDecoder(r.Body).Decode(&body)
			// Validate type
			if body.Type != "smb" && body.Type != "nfs" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "type must be smb or nfs"})
				return
			}
			// Validate name
			nameRe := regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
			if !nameRe.MatchString(body.Name) {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid name; use 1-32 characters [A-Za-z0-9_-]"})
				return
			}
			// Validate path under allowed roots
			roots := []string{"/srv", "/mnt"}
			if list, err := pools.ListPools(r.Context()); err == nil {
				for _, p := range list {
					if p.Mount != "" {
						roots = append(roots, p.Mount)
					}
				}
			}
			clean := filepath.Clean(body.Path)
			st := shares.NewStore(cfg.SharesPath)
			if !isUnderAllowed(clean, roots) {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "path not under allowed roots"})
				return
			}
			if fi, err := os.Stat(clean); err != nil || !fi.IsDir() {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "path does not exist or is not a directory"})
				return
			}
			// Collisions: name or path
			for _, ex := range st.List() {
				if ex.Name == body.Name {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "share name already exists"})
					return
				}
				if filepath.Clean(ex.Path) == clean {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "path already shared"})
					return
				}
			}
			if body.ID == "" {
				body.ID = body.Name
			}
			_ = st.Add(body)
			if body.Type == "smb" {
				// write smb snippet and reload
				_ = writeSmbShare(cfg.EtcDir, body)
				client := agentclient.New("/run/nos-agent.sock")
				_ = client.PostJSON(r.Context(), "/v1/service/reload", map[string]any{"name": "smb"}, nil)
			}
			if body.Type == "nfs" {
				_ = appendNfsExport(cfg.EtcDir, body)
				client := agentclient.New("/run/nos-agent.sock")
				_ = client.PostJSON(r.Context(), "/v1/service/reload", map[string]any{"name": "nfs"}, nil)
			}
			writeJSON(w, map[string]any{"ok": true})
		})
		pr.Delete("/api/shares/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			st := shares.NewStore(cfg.SharesPath)
			if sh, ok := st.GetByID(id); ok {
				if sh.Type == "smb" {
					_ = removeSmbShare(cfg.EtcDir, sh.Name)
				}
				if sh.Type == "nfs" {
					_ = removeNfsExport(cfg.EtcDir, sh.Path)
				}
			}
			_ = st.Delete(id)
			client := agentclient.New("/run/nos-agent.sock")
			_ = client.PostJSON(r.Context(), "/v1/service/reload", map[string]any{"name": "smb"}, nil)
			_ = client.PostJSON(r.Context(), "/v1/service/reload", map[string]any{"name": "nfs"}, nil)
			writeJSON(w, map[string]any{"ok": true})
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
			w.WriteHeader(http.StatusNotFound)
		})

		pr.Post("/api/apps/install", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				ID     string
				Config map[string]any
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ID == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "id required"})
				return
			}
			dir := filepath.Join(cfg.AppsInstallDir, body.ID)
			_ = os.MkdirAll(dir, 0o755)
			compose := apps.ComposeTemplate(body.ID)
			if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0o644); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			unit := apps.UnitTemplate(body.ID, dir)
			client := agentclient.New("/run/nos-agent.sock")
			_ = client.PostJSON(r.Context(), "/v1/systemd/install-app", map[string]any{"id": body.ID, "unit_text": unit}, nil)
			_ = client.PostJSON(r.Context(), "/v1/app/compose-up", map[string]any{"id": body.ID, "dir": dir}, nil)
			writeJSON(w, map[string]any{"ok": true})
		})

		pr.Post("/api/apps/uninstall", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				ID    string
				Force bool
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.ID == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "id required"})
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
		pr.Post("/api/firewall/plan", func(w http.ResponseWriter, r *http.Request) {
			var body struct{ Mode string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			rules := firewall.BuildRules(body.Mode)
			writeJSON(w, map[string]any{"rules": rules})
		})
		pr.Post("/api/firewall/apply", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Mode           string
				TwoFactorToken string
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if strings.ToLower(body.Mode) != "lan-only" {
				if s, ok := codec.DecodeFromRequest(r); !ok || !s.TwoFA {
					w.WriteHeader(http.StatusForbidden)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "2FA required"})
					return
				}
			}
			st, _ := firewall.Detect()
			if st.UFWPresent || st.FirewalldPresent {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "UFW or firewalld active"})
				return
			}
			rules := firewall.BuildRules(body.Mode)
			client := agentclient.New("/run/nos-agent.sock")
			var resp map[string]any
			if err := client.PostJSON(r.Context(), "/v1/firewall/apply", map[string]any{"ruleset_text": rules, "persist": true}, &resp); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		pr.Post("/api/pools/{id}/snapshots", func(w http.ResponseWriter, r *http.Request) {
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
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
