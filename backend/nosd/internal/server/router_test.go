package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"nithronos/backend/nosd/internal/config"
)

func TestHealth(t *testing.T) {
	r := NewRouter(config.FromEnv())
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["ok"] != true || body["version"] == "" {
		t.Fatalf("unexpected body: %v", body)
	}
}

func TestDisksShape(t *testing.T) {
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	cfg := config.FromEnv()
	r := NewRouter(cfg)

	// Login to get cookies
	loginBody := map[string]string{"email": "admin@example.com", "password": "admin123"}
	lb, _ := json.Marshal(loginBody)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(lb))
	loginRes := httptest.NewRecorder()
	r.ServeHTTP(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", loginRes.Code, loginRes.Body.String())
	}
	cookies := loginRes.Result().Cookies()
	var csrf string
	for _, c := range cookies {
		if c.Name == "nos_csrf" {
			csrf = c.Value
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/disks", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	res := httptest.NewRecorder()

	r.ServeHTTP(res, req)

	if res.Code != 200 {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if _, ok := body["disks"]; !ok {
		t.Fatalf("missing disks: %v", body)
	}
}

func TestLsblkFixture(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/lsblk.json")
	if err != nil {
		t.Skip("fixture not present")
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("fixture json invalid: %v", err)
	}
	if _, ok := body["blockdevices"]; !ok {
		t.Fatalf("fixture missing blockdevices")
	}
}
