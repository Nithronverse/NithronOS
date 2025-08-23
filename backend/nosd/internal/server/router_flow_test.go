package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
		if res.Code != http.StatusBadRequest {
			t.Fatalf("expired token expected 400, got %d", res.Code)
		}
		if !strings.Contains(res.Body.String(), "setup.otp.expired") {
			t.Fatalf("expected setup.otp.expired, got %s", res.Body.String())
		}
	}

	// create-admin (without totp)
	{
		t.Log("create-admin")
		req := httptest.NewRequest(http.MethodPost, "/api/setup/create-admin", bytes.NewBuffer(mustJSON(map[string]any{"username": "alice", "password": "StrongPassw0rd!", "enable_totp": false})))
		req.Header.Set("Authorization", "Bearer "+token)
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		if res.Code != http.StatusNoContent {
			t.Fatalf("create-admin: expected 204, got %d %s", res.Code, res.Body.String())
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

func TestSetupVerifyOTP_TypedErrors(t *testing.T) {
	cfg := config.Defaults()
	r := NewRouter(cfg)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/setup/verify-otp", bytes.NewBufferString(`{"otp":"1"}`)))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "setup.otp.invalid") {
		t.Fatalf("expected code setup.otp.invalid, got %s", res.Body.String())
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

func TestCreateAdmin_WriteFailThenRetrySameToken(t *testing.T) {
	// Prepare isolated users db and secret
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.key")
	usersPath := filepath.Join(dir, "users.json")
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(1 + i)
	}
	if err := os.WriteFile(secretPath, key, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NOS_SECRET_PATH", secretPath)
	t.Setenv("NOS_USERS_PATH", usersPath)
	cfg := config.FromEnv()
	r := NewRouter(cfg)

	// Issue a setup token directly
	sc := securecookie.New(key, nil)
	claims := map[string]any{"purpose": "setup", "exp": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)}
	tok, _ := sc.Encode("nos_setup", claims)

	// Simulate write failure
	t.Setenv("NOS_TEST_SIMULATE_WRITE_FAIL", "1")
	body := mustJSON(map[string]any{"username": "bob", "password": "StrongPassw0rd!"})
	req := httptest.NewRequest(http.MethodPost, "/api/setup/create-admin", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "setup.write_failed") {
		t.Fatalf("expected setup.write_failed, got %s", res.Body.String())
	}

	// Clear failure flag and retry with SAME token and a fresh request body
	t.Setenv("NOS_TEST_SIMULATE_WRITE_FAIL", "0")
	req2 := httptest.NewRequest(http.MethodPost, "/api/setup/create-admin", bytes.NewBuffer(body))
	req2.Header.Set("Authorization", "Bearer "+tok)
	res2 := httptest.NewRecorder()
	r.ServeHTTP(res2, req2)
	if res2.Code != http.StatusNoContent {
		t.Fatalf("retry expected 204, got %d (%s)", res2.Code, res2.Body.String())
	}
}

func TestSetupState_MissingEmptyInvalid(t *testing.T) {
    dir := t.TempDir()
    secretPath := filepath.Join(dir, "secret.key")
    usersPath := filepath.Join(dir, "users.json")
    firstbootPath := filepath.Join(dir, "firstboot.json")
    key := make([]byte, 32)
    for i := range key { key[i] = byte(1 + i) }
    if err := os.WriteFile(secretPath, key, 0o600); err != nil { t.Fatal(err) }
    t.Setenv("NOS_SECRET_PATH", secretPath)
    t.Setenv("NOS_USERS_PATH", usersPath)
    t.Setenv("NOS_FIRSTBOOT_PATH", firstbootPath)
    // Start with empty users.json
    if err := os.WriteFile(usersPath, []byte(""), 0o600); err != nil { t.Fatal(err) }
    cfg := config.FromEnv()
    r := NewRouter(cfg)

    // Missing users.json
    {
        res := httptest.NewRecorder()
        r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
        if res.Code != http.StatusOK { t.Fatalf("missing: expected 200, got %d", res.Code) }
        var st map[string]any
        _ = json.Unmarshal(res.Body.Bytes(), &st)
        if fb, _ := st["firstBoot"].(bool); !fb { t.Fatalf("missing: expected firstBoot=true, got %v", st) }
    }

    // Empty users.json
    if err := os.WriteFile(usersPath, []byte(""), 0o600); err != nil { t.Fatal(err) }
    {
        res := httptest.NewRecorder()
        r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
        if res.Code != http.StatusOK { t.Fatalf("empty: expected 200, got %d", res.Code) }
        var st map[string]any
        _ = json.Unmarshal(res.Body.Bytes(), &st)
        if fb, _ := st["firstBoot"].(bool); !fb { t.Fatalf("empty: expected firstBoot=true, got %v", st) }
    }

    // Invalid users.json
    if err := os.WriteFile(usersPath, []byte("{"), 0o600); err != nil { t.Fatal(err) }
    {
        res := httptest.NewRecorder()
        r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
        if res.Code != http.StatusOK { t.Fatalf("invalid: expected 200, got %d", res.Code) }
        var st map[string]any
        _ = json.Unmarshal(res.Body.Bytes(), &st)
        if fb, _ := st["firstBoot"].(bool); !fb { t.Fatalf("invalid: expected firstBoot=true, got %v", st) }
    }
}

func TestSetupTransition_DeleteUsersRestoresFirstBoot(t *testing.T) {
    dir := t.TempDir()
    secretPath := filepath.Join(dir, "secret.key")
    usersPath := filepath.Join(dir, "users.json")
    firstbootPath := filepath.Join(dir, "firstboot.json")
    key := make([]byte, 32)
    for i := range key { key[i] = byte(1 + i) }
    if err := os.WriteFile(secretPath, key, 0o600); err != nil { t.Fatal(err) }
    t.Setenv("NOS_SECRET_PATH", secretPath)
    t.Setenv("NOS_USERS_PATH", usersPath)
    t.Setenv("NOS_FIRSTBOOT_PATH", firstbootPath)
    // seed firstboot otp
    otp := "654321"
    fb := map[string]any{"otp": otp, "created_at": time.Now().UTC().Format(time.RFC3339), "used": false}
    if b, _ := json.MarshalIndent(fb, "", "  "); b != nil {
        if err := os.WriteFile(firstbootPath, b, 0o600); err != nil { t.Fatal(err) }
    }
    // high rate limits
    t.Setenv("NOS_RATE_LOGIN_PER_15M", "1000")
    t.Setenv("NOS_RATE_OTP_PER_MIN", "1000")

    cfg := config.FromEnv()
    r := NewRouter(cfg)

    // Initially: firstBoot=true
    {
        res := httptest.NewRecorder()
        r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
        if res.Code != http.StatusOK { t.Fatalf("initial: %d", res.Code) }
    }

    // Verify OTP to get token
    var token string
    {
        res := httptest.NewRecorder()
        r.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/setup/verify-otp", bytes.NewBuffer(mustJSON(map[string]string{"otp": otp}))))
        if res.Code != http.StatusOK { t.Fatalf("verify: %d", res.Code) }
        var out map[string]any
        _ = json.Unmarshal(res.Body.Bytes(), &out)
        token, _ = out["token"].(string)
        if token == "" { t.Fatal("missing token") }
    }

    // Create admin
    {
        req := httptest.NewRequest(http.MethodPost, "/api/setup/create-admin", bytes.NewBuffer(mustJSON(map[string]any{"username":"root","password":"StrongPassw0rd!"})))
        req.Header.Set("Authorization", "Bearer "+token)
        res := httptest.NewRecorder()
        r.ServeHTTP(res, req)
        if res.Code != http.StatusNoContent { t.Fatalf("create-admin: %d %s", res.Code, res.Body.String()) }
    }

    // Now gated: /state returns 410
    {
        res := httptest.NewRecorder()
        r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
        if res.Code != http.StatusGone { t.Fatalf("after setup: expected 410, got %d", res.Code) }
    }

    // Delete users.json; firstBoot should be true again
    _ = os.Remove(usersPath)
    {
        res := httptest.NewRecorder()
        r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/setup/state", nil))
        if res.Code != http.StatusOK { t.Fatalf("after delete users: %d", res.Code) }
        var st map[string]any
        _ = json.Unmarshal(res.Body.Bytes(), &st)
        if fb, _ := st["firstBoot"].(bool); !fb { t.Fatalf("after delete users: expected firstBoot=true, got %v", st) }
    }
}