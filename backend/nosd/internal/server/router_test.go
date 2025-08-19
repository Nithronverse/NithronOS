package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
	r := NewRouter(config.FromEnv())
	req := httptest.NewRequest(http.MethodGet, "/api/disks", nil)
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
	data, err := ioutil.ReadFile("internal/server/testdata/lsblk.json")
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
