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

func TestSnapshotCreate_RejectsInvalid(t *testing.T) {
	body := SnapshotCreateRequest{Path: "relative/not/allowed", Mode: "auto"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/snapshot/create", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	handleSnapshotCreate(rr, req)
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

func TestSnapshotList_RejectsInvalid(t *testing.T) {
	body := SnapshotListRequest{Path: "relative"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/snapshot/list", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	handleSnapshotList(rr, req)
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

func TestSnapshotCreate_Tar_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on windows")
	}
	// if tar missing in environment, skip
	if _, err := os.Stat("/bin/tar"); os.IsNotExist(err) {
		t.Skip("tar not available")
	}
	base := t.TempDir()
	// create sample file
	_ = os.WriteFile(filepath.Join(base, "file.txt"), []byte("hello"), 0o644)
	body := SnapshotCreateRequest{Path: base, Mode: "tar", Reason: "test"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/snapshot/create", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	handleSnapshotCreate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSnapshotCreate_Btrfs_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not supported on windows")
	}
	if _, err := exec.LookPath("btrfs"); err != nil {
		t.Skip("btrfs not available")
	}
	base := t.TempDir()
	// only run if the temp directory is on a btrfs subvolume
	if out, err := exec.Command("btrfs", "subvolume", "show", base).CombinedOutput(); err != nil || len(out) == 0 {
		t.Skip("not on btrfs; skipping")
	}
	body := SnapshotCreateRequest{Path: base, Mode: "btrfs", Reason: "test"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/snapshot/create", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	handleSnapshotCreate(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}
