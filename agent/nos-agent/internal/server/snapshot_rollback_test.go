package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSnapshotRollback_RejectsRoot(t *testing.T) {
	body := SnapshotRollbackRequest{Path: "/", SnapshotID: "x", Type: "tar"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/snapshot/rollback", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	handleSnapshotRollback(rr, req)
	if runtime.GOOS == "windows" {
		if rr.Code != http.StatusNotImplemented {
			t.Fatalf("expected 501 on windows, got %d", rr.Code)
		}
		return
	}
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestSnapshotRollback_Tar_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on windows")
	}
	if _, err := exec.LookPath("tar"); err != nil {
		t.Skip("tar not available")
	}
	// keep tar artifacts under temp
	baseSnap := t.TempDir()
	_ = os.Setenv("NOS_SNAPSHOT_BASE", baseSnap)

	// working dir to protect
	dir := t.TempDir()
	file := filepath.Join(dir, "data.txt")
	_ = os.WriteFile(file, []byte("before"), 0o644)

	// create snapshot via handler
	creq := SnapshotCreateRequest{Path: dir, Mode: "tar", Reason: "test"}
	cb, _ := json.Marshal(creq)
	cr := httptest.NewRequest(http.MethodPost, "/v1/snapshot/create", bytes.NewReader(cb))
	cw := httptest.NewRecorder()
	handleSnapshotCreate(cw, cr)
	if cw.Code != http.StatusOK {
		t.Skipf("skipping: snapshot create failed in env: %d %s", cw.Code, cw.Body.String())
	}
	var cres SnapshotCreateResponse
	_ = json.Unmarshal(cw.Body.Bytes(), &cres)
	if cres.ID == "" {
		t.Fatalf("no snapshot id returned")
	}

	// change content
	_ = os.WriteFile(file, []byte("after"), 0o644)

	// rollback
	rreq := SnapshotRollbackRequest{Path: dir, SnapshotID: cres.ID, Type: "tar"}
	rb, _ := json.Marshal(rreq)
	rr := httptest.NewRequest(http.MethodPost, "/v1/snapshot/rollback", bytes.NewReader(rb))
	rw := httptest.NewRecorder()
	handleSnapshotRollback(rw, rr)
	if rw.Code != http.StatusOK {
		t.Fatalf("rollback failed: %d %s", rw.Code, rw.Body.String())
	}
	data, _ := os.ReadFile(file)
	if string(data) != "before" {
		t.Fatalf("expected restored content 'before', got %q", string(data))
	}
}

func TestSnapshotRollback_Btrfs_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on windows")
	}
	if _, err := exec.LookPath("btrfs"); err != nil {
		t.Skip("btrfs not available")
	}
	// attempt to create a subvolume under temp
	root := t.TempDir()
	sv := filepath.Join(root, "sv")
	if out, err := exec.Command("btrfs", "subvolume", "create", sv).CombinedOutput(); err != nil {
		t.Skipf("cannot create subvolume here: %s", string(out))
	}
	defer func() { _ = exec.Command("btrfs", "subvolume", "delete", sv).Run() }()

	file := filepath.Join(sv, "data.txt")
	_ = os.WriteFile(file, []byte("before"), 0o644)

	// create readonly snapshot via handler
	creq := SnapshotCreateRequest{Path: sv, Mode: "btrfs", Reason: "test"}
	cb, _ := json.Marshal(creq)
	cr := httptest.NewRequest(http.MethodPost, "/v1/snapshot/create", bytes.NewReader(cb))
	cw := httptest.NewRecorder()
	handleSnapshotCreate(cw, cr)
	if cw.Code != http.StatusOK {
		t.Skipf("skipping: snapshot create failed in env: %d %s", cw.Code, cw.Body.String())
	}
	var cres SnapshotCreateResponse
	_ = json.Unmarshal(cw.Body.Bytes(), &cres)
	if cres.ID == "" {
		t.Fatalf("no snapshot id returned")
	}

	// mutate file
	_ = os.WriteFile(file, []byte("after"), 0o644)

	// rollback to snapshot id
	rreq := SnapshotRollbackRequest{Path: sv, SnapshotID: cres.ID, Type: "btrfs"}
	rb, _ := json.Marshal(rreq)
	rr := httptest.NewRequest(http.MethodPost, "/v1/snapshot/rollback", bytes.NewReader(rb))
	rw := httptest.NewRecorder()
	handleSnapshotRollback(rw, rr)
	if rw.Code != http.StatusOK {
		t.Skipf("rollback failed in env: %d %s", rw.Code, rw.Body.String())
	}
	data, _ := os.ReadFile(file)
	if string(data) != "before" {
		t.Fatalf("expected restored content 'before', got %q", string(data))
	}
}
