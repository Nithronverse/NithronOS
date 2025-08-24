package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	userstore "nithronos/backend/nosd/internal/auth/store"
	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/internal/fsatomic"
	"nithronos/backend/nosd/internal/server"
	firstboot "nithronos/backend/nosd/internal/setup/firstboot"
)

func main() {
	cfg := config.Load("/etc/nos/config.yaml")
	server.SetRuntimeCORSOrigin(cfg.CORSOrigin)
	server.SetRuntimeTrustProxy(cfg.TrustProxy)
	server.SetLogLevel(cfg.LogLevel)
	ensureSecret(cfg.SecretPath)
	ensureAgentToken("/etc/nos/agent-token")
	// First-boot OTP: ensure state dir and reuse or create
	_ = os.MkdirAll(filepath.Dir(cfg.FirstBootPath), 0o750)
	fb := firstboot.New(cfg.FirstBootPath)
	if st, reused, err := fb.NewOrReuse(15*time.Minute, generateOTP6); err == nil && st != nil {
		msg := "First-boot OTP: " + st.OTP + " (valid 15m)"
		if reused {
			msg = "Using existing first-boot OTP: " + st.OTP + " (valid 15m)"
		}
		fmt.Println(msg)
		server.Logger(cfg).Info().Msg(msg)
	}
	r := server.NewRouter(cfg)

	addr := cfg.Bind
	server.Logger(cfg).Info().Msgf("nosd listening on http://%s", addr)

	go func() {
		// SIGHUP hot reload (Unix only)
		if runtime.GOOS != "windows" {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch)
			for range ch {
				old := cfg
				cfg = config.Load("/etc/nos/config.yaml")
				// Apply safe fields
				server.SetRuntimeCORSOrigin(cfg.CORSOrigin)
				server.SetRuntimeTrustProxy(cfg.TrustProxy)
				server.SetLogLevel(cfg.LogLevel)
				logConfigDiff(old, cfg)
			}
		}
	}()

	if err := http.ListenAndServe(addr, r); err != nil {
		server.Logger(cfg).Fatal().Err(err).Msg("server exited")
	}
}

func logConfigDiff(old, cur config.Config) {
	// minimal diff of hot-reloadable fields
	if old.CORSOrigin != cur.CORSOrigin {
		server.Logger(cur).Info().Str("event", "config.reload").Str("field", "cors.origin").Str("old", old.CORSOrigin).Str("new", cur.CORSOrigin).Msg("")
	}
	if old.TrustProxy != cur.TrustProxy {
		server.Logger(cur).Info().Str("event", "config.reload").Str("field", "trustProxy").Bool("old", old.TrustProxy).Bool("new", cur.TrustProxy).Msg("")
	}
	if old.LogLevel != cur.LogLevel {
		server.Logger(cur).Info().Str("event", "config.reload").Str("field", "logLevel").Str("old", old.LogLevel.String()).Str("new", cur.LogLevel.String()).Msg("")
	}
}

func ensureSecret(path string) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err == nil {
		return
	}
	// create parent dirs
	if dir := dirOf(path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return
	}
	_ = os.WriteFile(path, b, 0o600)
	fp := hex.EncodeToString(b)
	if len(fp) > 8 {
		fp = fp[:8]
	}
	fmt.Printf("generated secret key at %s (fp=%s)\n", path, fp)
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			if i == 0 {
				return "/"
			}
			return p[:i]
		}
	}
	return "."
}

// ensureFirstBootOTP initializes or refreshes the first-boot OTP if no admin exists.
func ensureFirstBootOTP(cfg config.Config) {
	us, err := userstore.New(cfg.UsersPath)
	if err != nil {
		return
	}
	if us.HasAdmin() {
		// Flip state to used so setup endpoints 410 Gone
		type fbState struct {
			OTP       string `json:"otp"`
			CreatedAt string `json:"created_at"`
			Used      bool   `json:"used"`
		}
		if b, err := os.ReadFile(cfg.FirstBootPath); err == nil {
			var st fbState
			if json.Unmarshal(b, &st) == nil && !st.Used {
				st.Used = true
				_ = os.MkdirAll(filepath.Dir(cfg.FirstBootPath), 0o755)
				_ = fsatomic.SaveJSON(context.TODO(), cfg.FirstBootPath, st, 0o600)
			}
		}
		return
	}
	type fbState struct {
		OTP       string `json:"otp"`
		CreatedAt string `json:"created_at"`
		Used      bool   `json:"used"`
	}
	var st fbState
	if b, err := os.ReadFile(cfg.FirstBootPath); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	valid := false
	if st.OTP != "" && !st.Used {
		if t, err := time.Parse(time.RFC3339, st.CreatedAt); err == nil {
			if time.Since(t) < 15*time.Minute {
				valid = true
			}
		}
	}
	if !valid {
		st.OTP = generateOTP6()
		st.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		st.Used = false
		_ = os.MkdirAll(filepath.Dir(cfg.FirstBootPath), 0o755)
		_ = fsatomic.SaveJSON(context.TODO(), cfg.FirstBootPath, st, 0o600)
	}
	msg := fmt.Sprintf("First-boot OTP: %s (valid 15m)", st.OTP)
	fmt.Println(msg)
	server.Logger(cfg).Info().Msg(msg)
}

func generateOTP6() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "000000"
	}
	n := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) % 1000000
	return fmt.Sprintf("%06d", n)
}

func ensureAgentToken(path string) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err == nil {
		return
	}
	if dir := dirOf(path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return
	}
	_ = os.WriteFile(path, b[:], 0o600)
}
