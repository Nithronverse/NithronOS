package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

type fakeAgentUpdates struct {
	calls []struct {
		path string
		body map[string]any
	}
	fail map[string]error
}

func (f *fakeAgentUpdates) PostJSON(_ context.Context, path string, body any, v any) error {
	bmap, _ := body.(map[string]any)
	f.calls = append(f.calls, struct {
		path string
		body map[string]any
	}{path, bmap})
	if err, ok := f.fail[path]; ok && err != nil {
		return err
	}
	switch path {
	case "/v1/snapshot/create":
		if v != nil {
			// return ok with synthetic id/type/location
			_ = json.Unmarshal([]byte(`{"ok":true,"id":"snap-`+time.Now().UTC().Format("150405")+`","type":"tar","location":"/var/lib/nos/snapshots/x"}`), v)
		}
	case "/v1/updates/apply":
		if v != nil {
			_ = json.Unmarshal([]byte(`{"ok":true}`), v)
		}
	case "/v1/snapshot/rollback":
		if v != nil {
			_ = json.Unmarshal([]byte(`{"ok":true}`), v)
		}
	}
	return nil
}

func TestUpdatesApply_HappyPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	// temp snapdb dir
	dir := t.TempDir()
	_ = os.Setenv("NOS_SNAPDB_DIR", dir)
	defer os.Unsetenv("NOS_SNAPDB_DIR")

	cfg := config.FromEnv()
	r := NewRouter(cfg)
	prev := handlers.AgentClientFactory
	fake := &fakeAgentUpdates{}
	handlers.AgentClientFactory = func() handlers.AgentPoster { return fake }
	defer func() { handlers.AgentClientFactory = prev }()

	body := map[string]any{"packages": []string{"nosd"}, "snapshot": true, "confirm": "yes"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", bytes.NewReader(b))
	// auth bypass via test helper withAuth from shares tests
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
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	// check tx saved
	items, err := snapdb.ListRecent(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) == 0 || items[len(items)-1].Success == nil || !*items[len(items)-1].Success {
		t.Fatalf("expected success record, got %+v", items)
	}
	// verify two snapshot/create calls (roots default: /srv and /mnt)
	snaps := 0
	for _, c := range fake.calls {
		if c.path == "/v1/snapshot/create" {
			snaps++
		}
	}
	if snaps < 2 {
		t.Fatalf("expected at least 2 snapshot creates, got %d: %+v", snaps, fake.calls)
	}
}

func TestUpdatesApply_SnapshotError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	dir := t.TempDir()
	_ = os.Setenv("NOS_SNAPDB_DIR", dir)
	defer os.Unsetenv("NOS_SNAPDB_DIR")

	cfg := config.FromEnv()
	r := NewRouter(cfg)
	prev := handlers.AgentClientFactory
	fake := &fakeAgentUpdates{fail: map[string]error{"/v1/snapshot/create": fmt.Errorf("boom")}}
	handlers.AgentClientFactory = func() handlers.AgentPoster { return fake }
	defer func() { handlers.AgentClientFactory = prev }()

	body := map[string]any{"packages": []string{"nosd"}, "snapshot": true, "confirm": "yes"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/updates/apply", bytes.NewReader(b))
	cookies, csrf := withAuth(t, r)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", w.Code, w.Body.String())
	}
	items, err := snapdb.ListRecent(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) == 0 || items[len(items)-1].Success == nil || *items[len(items)-1].Success {
		t.Fatalf("expected failure record, got %+v", items)
	}
}

func TestUpdatesRollback_HappyPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip on windows")
	}
	dir := t.TempDir()
	_ = os.Setenv("NOS_SNAPDB_DIR", dir)
	defer os.Unsetenv("NOS_SNAPDB_DIR")

	// seed a completed tx
	t0 := time.Now().UTC()
	ok := true
	seed := snapdb.UpdateTx{TxID: "seed-tx", StartedAt: t0.Add(-time.Hour), FinishedAt: &t0, Packages: []string{"nosd"}, Reason: "pre-update", Targets: []snapdb.SnapshotTarget{
		{ID: "id1", Path: "/srv/a", Type: "tar", Location: "/var/lib/nos/snapshots/a/id1.tar.gz", CreatedAt: t0.Add(-time.Hour)},
		{ID: "id2", Path: "/srv/b", Type: "tar", Location: "/var/lib/nos/snapshots/b/id2.tar.gz", CreatedAt: t0.Add(-55 * time.Minute)},
	}, Success: &ok}
	_ = snapdb.Append(seed)

	cfg := config.FromEnv()
	r := NewRouter(cfg)
	prev := handlers.AgentClientFactory
	fake := &fakeAgentUpdates{}
	handlers.AgentClientFactory = func() handlers.AgentPoster { return fake }
	defer func() { handlers.AgentClientFactory = prev }()

	body := map[string]any{"tx_id": "seed-tx", "confirm": "yes"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/updates/rollback", bytes.NewReader(b))
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
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	items, err := snapdb.ListRecent(10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("no records found")
	}
	first := items[0]
	if first.Reason != "rollback" || first.Success == nil || !*first.Success {
		t.Fatalf("expected most recent to be rollback success, got %+v", first)
	}
	// verify agent rollback called for each target
	rb := 0
	for _, c := range fake.calls {
		if c.path == "/v1/snapshot/rollback" {
			rb++
		}
	}
	if rb != len(seed.Targets) {
		t.Fatalf("expected %d rollback calls, got %d", len(seed.Targets), rb)
	}
}
