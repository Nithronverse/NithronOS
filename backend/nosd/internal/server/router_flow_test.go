package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	userstore "nithronos/backend/nosd/internal/auth/store"
	"nithronos/backend/nosd/internal/config"

	"github.com/gorilla/securecookie"
	"github.com/pquerna/otp/totp"
)

func TestSetupFullFlowAnd410(t *testing.T) {
	// temp state
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.key")
	firstbootPath := filepath.Join(dir, "firstboot.json")
	usersPath := filepath.Join(dir, "users.json")
	// seed secret
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(1 + i)
	}
	if err := os.WriteFile(secretPath, key, 0o600); err != nil {
		t.Fatal(err)
	}
	// seed empty users file
	if err := os.WriteFile(usersPath, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	// seed firstboot otp
	otp := "123456"
	fb := map[string]any{"otp": otp, "created_at": time.Now().UTC().Format(time.RFC3339), "used": false}
	if b, _ := json.MarshalIndent(fb, "", "  "); b != nil {
		if err := os.WriteFile(firstbootPath, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.Setenv("NOS_SECRET_PATH", secretPath)
	_ = os.Setenv("NOS_FIRSTBOOT_PATH", firstbootPath)
	_ = os.Setenv("NOS_USERS_PATH", usersPath)
	// Relax rate limits for this flow test to avoid incidental 429s
	_ = os.Setenv("NOS_RATE_LOGIN_PER_15M", "1000")
	_ = os.Setenv("NOS_RATE_OTP_PER_MIN", "1000")

	cfg := config.FromEnv()
	r := NewRouter(cfg)

	// state
	t.Log("state")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
	if res.Code != 200 {
		t.Fatalf("state: expected 200, got %d", res.Code)
	}

	// verify-otp
	var token string
	{
		t.Log("verify-otp")
		body := bytes.NewBuffer(mustJSON(map[string]string{"otp": otp}))
		res := httptest.NewRecorder()
		r.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/setup/verify-otp", body))
		if res.Code != 200 {
			t.Fatalf("verify-otp: %d", res.Code)
		}
		var out map[string]any
		_ = json.Unmarshal(res.Body.Bytes(), &out)
		token, _ = out["token"].(string)
		if token == "" {
			t.Fatal("missing token")
		}
	}

	// expired token should fail
	{
		t.Log("expired-token")
		sc := securecookie.New(key, nil)
		claims := map[string]any{"purpose": "setup", "exp": time.Now().Add(-1 * time.Minute).UTC().Format(time.RFC3339)}
		expTok, _ := sc.Encode("nos_setup", claims)
		req := httptest.NewRequest(http.MethodPost, "/api/setup/create-admin", bytes.NewBuffer(mustJSON(map[string]any{"username": "alice", "password": "StrongPassw0rd!"})))
		req.Header.Set("Authorization", "Bearer "+expTok)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("expired token expected 401, got %d", res.Code)
		}
	}

	// create-admin (without totp)
	{
		t.Log("create-admin")
		req := httptest.NewRequest(http.MethodPost, "/api/setup/create-admin", bytes.NewBuffer(mustJSON(map[string]any{"username": "alice", "password": "StrongPassw0rd!", "enable_totp": false})))
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("create-admin: %d %s", res.Code, res.Body.String())
		}
	}

	// state now 410
	{
		t.Log("state-410")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
		if res.Code != http.StatusGone {
			t.Fatalf("state after setup: expected 410, got %d", res.Code)
		}
	}

	// login to get session and csrf
	var cookies []*http.Cookie
	var csrf string
	{
		t.Log("login-1")
		lb := mustJSON(map[string]any{"username": "alice", "password": "StrongPassw0rd!"})
		res := httptest.NewRecorder()
		r.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(lb)))
		if res.Code != 200 {
			t.Fatalf("login: %d %s", res.Code, res.Body.String())
		}
		cookies = res.Result().Cookies()
		for _, c := range cookies {
			if c.Name == "nos_csrf" {
				csrf = c.Value
			}
		}
		// verify cookie presence
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		if uid, ok := decodeSessionUID(req, cfg); !ok || uid == "" {
			t.Fatalf("nos_session did not decode; cookies=%v", cookies)
		}
	}

	// sessions list
	{
		t.Log("sessions-list")
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("sessions list: %d", res.Code)
		}
		var arr []map[string]any
		_ = json.Unmarshal(res.Body.Bytes(), &arr)
		if len(arr) == 0 {
			t.Fatalf("expected at least one session")
		}
	}

	// revoke current session
	{
		t.Log("revoke-current")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/sessions/revoke", bytes.NewReader(mustJSON(map[string]string{"scope": "current"})))
		for _, c := range cookies {
			req.AddCookie(c)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("revoke current: %d", res.Code)
		}
		// me should now be unauthorized
		res2 := httptest.NewRecorder()
		r.ServeHTTP(res2, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
		if res2.Code != http.StatusUnauthorized {
			t.Fatalf("me after revoke current: %d", res2.Code)
		}
	}

	// enroll
	{
		t.Log("enroll")
		req := httptest.NewRequest(http.MethodGet, "/api/auth/totp/enroll", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("enroll: %d", res.Code)
		}
	}

	// fetch user to get secret and generate code
	secCode := func() string {
		t.Log("get-secret")
		us, _ := userstore.New(usersPath)
		u, _ := us.FindByUsername("alice")
		if u.TOTPEnc == "" {
			t.Fatal("expected encrypted secret after enroll")
		}
		pt, err := decryptWithSecretKey(secretPath, u.TOTPEnc)
		if err != nil {
			t.Fatalf("decrypt secret: %v", err)
		}
		code, _ := totp.GenerateCode(string(pt), time.Now())
		return code
	}()

	// verify
	{
		t.Log("verify")
		vb := mustJSON(map[string]string{"code": secCode})
		req := httptest.NewRequest(http.MethodPost, "/api/auth/totp/verify", bytes.NewReader(vb))
		for _, c := range cookies {
			req.AddCookie(c)
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("verify totp: %d %s", res.Code, res.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(res.Body.Bytes(), &out)
		if _, ok := out["recovery_codes"].([]any); !ok {
			t.Fatal("missing recovery codes")
		}
	}

	// login with code now required
	{
		t.Log("login-2")
		code, _ := totp.GenerateCode(secCodeSecret(t, secretPath, usersPath), time.Now())
		lb := mustJSON(map[string]any{"username": "alice", "password": "StrongPassw0rd!", "code": code})
		res := httptest.NewRecorder()
		r.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(lb)))
		if res.Code != 200 {
			t.Fatalf("login with code: %d", res.Code)
		}
		cookies = res.Result().Cookies()
		csrf = ""
		for _, c := range cookies {
			if c.Name == "nos_csrf" {
				csrf = c.Value
			}
		}
	}

	// me
	{
		t.Log("me")
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != 200 {
			t.Fatalf("me: %d", res.Code)
		}
	}

	// logout
	{
		t.Log("logout")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))
		if res.Code != http.StatusNoContent {
			t.Fatalf("logout: %d", res.Code)
		}
	}

	// me unauthorized
	{
		t.Log("me-unauth")
		res := httptest.NewRecorder()
		r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("me after logout: %d", res.Code)
		}
	}
}

func secCodeSecret(t *testing.T, secretPath, usersPath string) string {
	us, _ := userstore.New(usersPath)
	u, _ := us.FindByUsername("alice")
	pt, err := decryptWithSecretKey(secretPath, u.TOTPEnc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	return string(pt)
}
