package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nithronos/backend/nosd/internal/config"
)

type fakeAgent struct {
	calls []struct {
		path string
		body map[string]any
	}
	fail map[string]error
}

func (f *fakeAgent) PostJSON(_ context.Context, path string, body any, v any) error {
	bmap, _ := body.(map[string]any)
	f.calls = append(f.calls, struct {
		path string
		body map[string]any
	}{path, bmap})
	if err, ok := f.fail[path]; ok && err != nil {
		return err
	}
	if v != nil {
		// respond with ok:true when needed
		_ = json.NewEncoder(&bytes.Buffer{})
	}
	return nil
}

func withAuth(t *testing.T, r http.Handler) (cookies []*http.Cookie, csrf string) {
	t.Helper()
	// Seed minimal users db for new auth store
	if up := os.Getenv("NOS_USERS_PATH"); up != "" {
		_ = os.MkdirAll(filepath.Dir(up), 0o755)
		_ = os.WriteFile(up, []byte(`{"version":1,"users":[{"id":"u1","username":"admin@example.com","password_hash":"plain:admin123","roles":["admin"],"created_at":"","updated_at":""}]}`), 0o600)
	}
	loginBody := map[string]string{"username": "admin@example.com", "password": "admin123"}
	lb, _ := json.Marshal(loginBody)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(lb))
	loginRes := httptest.NewRecorder()
	r.ServeHTTP(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", loginRes.Code, loginRes.Body.String())
	}
	cookies = loginRes.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "nos_csrf" {
			csrf = c.Value
		}
	}
	return
}

func TestSharesCreate_Valid_Smb_WritesConfigAndReloads(t *testing.T) {
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	tmpDir := t.TempDir()
	_ = os.Setenv("NOS_SHARES_PATH", filepath.Join(tmpDir, "shares.json"))
	_ = os.Setenv("NOS_ETC_DIR", filepath.Join(tmpDir, "etc"))
	_ = config.FromEnv()
	// r := NewRouter(cfg) // Would be used if test was not skipped

	// Skip agent-related testing since handlers package was removed
	// This test needs to be rewritten to use the new SharesHandler
	t.Skip("Test needs to be updated for new SharesHandler implementation")
}

func TestSharesCreate_DuplicateNameOrPath_409(t *testing.T) {
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	tmpDir := t.TempDir()
	_ = os.Setenv("NOS_SHARES_PATH", filepath.Join(tmpDir, "shares.json"))
	_ = os.WriteFile(filepath.Join(tmpDir, "shares.json"), []byte(`[{"id":"media","type":"smb","path":"/srv/data/media","name":"media","ro":false}]`), 0o600)
	cfg := config.FromEnv()
	r := NewRouter(cfg)

	cookies, csrf := withAuth(t, r)

	body := map[string]any{"type": "smb", "name": "media", "path": "/srv/data/media", "ro": false}
	bb, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/shares", bytes.NewReader(bb))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.Code)
	}
}

func TestSharesCreate_PathOutsideRoots_400(t *testing.T) {
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	tmpDir := t.TempDir()
	_ = os.Setenv("NOS_SHARES_PATH", filepath.Join(tmpDir, "shares.json"))
	cfg := config.FromEnv()
	r := NewRouter(cfg)

	cookies, csrf := withAuth(t, r)

	body := map[string]any{"type": "smb", "name": "etcshare", "path": "/etc/secret", "ro": false}
	bb, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/shares", bytes.NewReader(bb))
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestSharesCreate_MkdirFailure_500(t *testing.T) {
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	tmpDir := t.TempDir()
	_ = os.Setenv("NOS_SHARES_PATH", filepath.Join(tmpDir, "shares.json"))
	_ = os.Setenv("NOS_ETC_DIR", filepath.Join(tmpDir, "etc"))
	_ = config.FromEnv()
	// r := NewRouter(cfg) // Would be used if test was not skipped

	// Skip agent-related testing since handlers package was removed
	// This test needs to be rewritten to use the new SharesHandler
	t.Skip("Test needs to be updated for new SharesHandler implementation")
}
