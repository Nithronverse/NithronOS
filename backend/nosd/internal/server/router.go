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

	r.Get("/api/disks", func(w http.ResponseWriter, r *http.Request) {
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

	r.Get("/api/pools", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{})
	})

	r.Get("/api/shares", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{})
	})

	r.Get("/api/apps", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []map[string]string{{"id": "jellyfin", "status": "not_installed"}})
	})

	r.Get("/api/remote/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"mode": "lan-only", "https": true})
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
