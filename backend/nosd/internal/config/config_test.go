package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestYAMLAndEnvPrecedence(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	data := []byte("" +
		"http:\n  bind: 127.0.0.1:9999\n" +
		"cors:\n  origin: http://example.com\n" +
		"rate:\n  otpPerMin: 7\n  loginPer15m: 9\n  otpWindowSec: 10\n  loginWindowSec: 11\n" +
		"trustProxy: true\n" +
		"logging:\n  level: debug\n" +
		"sessions:\n  accessTTL: 20m\n  refreshTTL: 100h\n" +
		"metrics:\n  enabled: true\n  pprof: true\n")
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	// baseline from file
	cfg := Load(cfgPath)
	if cfg.Bind != "127.0.0.1:9999" {
		t.Fatalf("bind from yaml: %s", cfg.Bind)
	}
	if cfg.CORSOrigin != "http://example.com" {
		t.Fatalf("cors from yaml: %s", cfg.CORSOrigin)
	}
	if !cfg.TrustProxy {
		t.Fatalf("trustProxy from yaml")
	}
	if cfg.RateOTPPerMin != 7 || cfg.RateLoginPer15m != 9 {
		t.Fatalf("rate from yaml: %v %v", cfg.RateOTPPerMin, cfg.RateLoginPer15m)
	}
	if cfg.RateOTPWindowSec != 10 || cfg.RateLoginWindowSec != 11 {
		t.Fatalf("rate win from yaml: %v %v", cfg.RateOTPWindowSec, cfg.RateLoginWindowSec)
	}
	if cfg.LogLevel.String() != "debug" {
		t.Fatalf("loglevel from yaml: %s", cfg.LogLevel)
	}
	if cfg.SessionAccessTTLSeconds != 1200 {
		t.Fatalf("access ttl: %d", cfg.SessionAccessTTLSeconds)
	}
	if cfg.SessionRefreshTTLSeconds != 360000 {
		t.Fatalf("refresh ttl: %d", cfg.SessionRefreshTTLSeconds)
	}
	if !cfg.MetricsEnabled || !cfg.PprofEnabled {
		t.Fatalf("metrics toggles")
	}

	// env overrides file
	t.Setenv("NOS_HTTP_BIND", "0.0.0.0:8080")
	t.Setenv("NOS_CORS_ORIGIN", "http://override")
	t.Setenv("NOS_TRUST_PROXY", "false")
	t.Setenv("NOS_RATE_OTP_PER_MIN", "3")
	t.Setenv("NOS_RATE_LOGIN_PER_15M", "4")
	t.Setenv("NOS_RATE_OTP_WINDOW_SEC", "20")
	t.Setenv("NOS_RATE_LOGIN_WINDOW_SEC", "30")
	t.Setenv("NOS_LOG", "warn")
	t.Setenv("NOS_SESSION_ACCESS_TTL", "30m")
	t.Setenv("NOS_SESSION_REFRESH_TTL", "200h")
	t.Setenv("NOS_METRICS", "0")
	t.Setenv("NOS_PPROF", "1")

	cfg2 := Load(cfgPath)
	if cfg2.Bind != "0.0.0.0:8080" {
		t.Fatalf("bind env override: %s", cfg2.Bind)
	}
	if cfg2.CORSOrigin != "http://override" {
		t.Fatalf("cors env override: %s", cfg2.CORSOrigin)
	}
	if cfg2.TrustProxy {
		t.Fatalf("trustProxy env override should be false")
	}
	if cfg2.RateOTPPerMin != 3 || cfg2.RateLoginPer15m != 4 {
		t.Fatalf("rate env override")
	}
	if cfg2.RateOTPWindowSec != 20 || cfg2.RateLoginWindowSec != 30 {
		t.Fatalf("rate window env override")
	}
	if cfg2.LogLevel.String() != "warn" {
		t.Fatalf("log env override: %s", cfg2.LogLevel)
	}
	if cfg2.SessionAccessTTLSeconds != 1800 {
		t.Fatalf("access env override: %d", cfg2.SessionAccessTTLSeconds)
	}
	if cfg2.SessionRefreshTTLSeconds != 720000 {
		t.Fatalf("refresh env override: %d", cfg2.SessionRefreshTTLSeconds)
	}
	if cfg2.MetricsEnabled {
		t.Fatalf("metrics should be disabled by env")
	}
	if !cfg2.PprofEnabled {
		t.Fatalf("pprof should be enabled by env")
	}
}
