package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"

	"nithronos/backend/nosd/handlers"
	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/pkg/snapdb"
)

type fakeAgentPlan struct{}

func (f *fakeAgentPlan) PostJSON(_ context.Context, path string, body any, v any) error {
	switch path {
	case "/v1/updates/plan":
		if v != nil {
			_ = json.Unmarshal([]byte(`{"updates":[{"name":"nosd","current":"0.1.0","candidate":"0.2.0"}],"reboot_required":false,"raw":""}`), v)
		}
	case "/v1/snapshot/prune":
		if v != nil {
			_ = json.Unmarshal([]byte(`{"ok":true,"pruned":{}}`), v)
		}
	}
	return nil
}

func TestUpdatesCheck_ReturnsPlanAndRoots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	cfg := config.FromEnv()
	r := NewRouter(cfg)
	prev := handlers.AgentClientFactory
	handlers.AgentClientFactory = func() handlers.AgentPoster { return &fakeAgentPlan{} }
	defer func() { handlers.AgentClientFactory = prev }()

	req := httptest.NewRequest(http.MethodGet, "/api/updates/check", nil)
	cookies, csrf := withAuth(t, r)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", w.Code, w.Body.String())
	}
}

func TestSnapshotsRecent_AndByTx(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	dir := t.TempDir()
	_ = os.Setenv("NOS_SNAPDB_DIR", dir)
	defer os.Unsetenv("NOS_SNAPDB_DIR")
	now := time.Now().UTC()
	ok := true
	seed := snapdb.UpdateTx{TxID: "tx-a", StartedAt: now, FinishedAt: &now, Packages: []string{"nosd"}, Reason: "pre-update", Success: &ok}
	_ = snapdb.Append(seed)

	cfg := config.FromEnv()
	r := NewRouter(cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/snapshots/recent", nil)
	cookies, csrf := withAuth(t, r)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("recent want 200 got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/snapshots/tx-a", nil)
	for _, c := range cookies {
		req2.AddCookie(c)
	}
	if csrf != "" {
		req2.Header.Set("X-CSRF-Token", csrf)
	}
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("bytx want 200 got %d", w2.Code)
	}
}

func TestSnapshotsPruneProxy_OK(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	_ = os.Setenv("NOS_USERS_PATH", "../../../../devdata/users.json")
	cfg := config.FromEnv()
	r := NewRouter(cfg)
	prev := handlers.AgentClientFactory
	handlers.AgentClientFactory = func() handlers.AgentPoster { return &fakeAgentPlan{} }
	defer func() { handlers.AgentClientFactory = prev }()
	body := map[string]any{"keep_per_target": 5}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/snapshots/prune", bytes.NewReader(b))
	cookies, csrf := withAuth(t, r)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", w.Code, w.Body.String())
	}
}
