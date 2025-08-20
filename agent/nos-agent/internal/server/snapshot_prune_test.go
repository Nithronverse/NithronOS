package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSnapshotPrune_TarDir_PrunesToKeep(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on windows")
	}
	// create a tar snapshot dir with 7 files
	dir := t.TempDir()
	for i := 1; i <= 7; i++ {
		name := filepath.Join(dir, time.Unix(int64(i), 0).Format("20060102-150405")+".tar.gz")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		// ensure mtime increasing
		_ = os.Chtimes(name, time.Unix(int64(i), 0), time.Unix(int64(i), 0))
	}
	// call handler with keep_per_target=5 and explicit path
	body := SnapshotPruneRequest{KeepPerTarget: 5, Paths: []string{dir}}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/snapshot/prune", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	handleSnapshotPrune(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	// verify only 5 tar.gz remain
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	cnt := 0
	for _, e := range ents {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".gz" {
			cnt++
		}
	}
	if cnt != 5 {
		t.Fatalf("expected 5 remaining, got %d", cnt)
	}
}
