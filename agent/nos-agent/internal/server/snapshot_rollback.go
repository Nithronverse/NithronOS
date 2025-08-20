package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type SnapshotRollbackRequest struct {
	Path         string   `json:"path"`
	SnapshotID   string   `json:"snapshot_id"`
	Type         string   `json:"type"` // btrfs|tar
	StopServices []string `json:"stop_services"`
}

func handleSnapshotRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runtime.GOOS == "windows" {
		writeErr(w, http.StatusNotImplemented, "not supported on windows")
		return
	}
	var req SnapshotRollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Path == "/" {
		writeErr(w, http.StatusBadRequest, "refuse to operate on rootfs")
		return
	}
	if req.Path == "" || !strings.HasPrefix(req.Path, "/") || req.SnapshotID == "" {
		writeErr(w, http.StatusBadRequest, "missing fields or invalid path")
		return
	}
	if fi, err := os.Stat(req.Path); err != nil || !fi.IsDir() {
		writeErr(w, http.StatusBadRequest, "path must exist and be a directory")
		return
	}
	// stop services if requested
	stopped := []string{}
	for _, s := range req.StopServices {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		cmd := exec.Command("systemctl", "stop", s)
		if out, err := cmd.CombinedOutput(); err != nil {
			logAuthPriv(fmt.Sprintf("rollback: stop %s failed: %s", s, string(out)))
		} else {
			stopped = append(stopped, s)
		}
	}
	defer func() {
		for _, s := range stopped {
			cmd := exec.Command("systemctl", "start", s)
			if out, err := cmd.CombinedOutput(); err != nil {
				logAuthPriv(fmt.Sprintf("rollback: start %s failed: %s", s, string(out)))
			}
		}
	}()

	ts := time.Now().UTC().Format("20060102-150405")
	rbDir := filepath.Join(req.Path, ".rollback")
	_ = os.MkdirAll(rbDir, 0o700)

	switch strings.ToLower(req.Type) {
	case "btrfs":
		// ensure path is a subvolume
		if _, err := exec.Command("btrfs", "subvolume", "show", req.Path).CombinedOutput(); err != nil {
			writeErr(w, http.StatusBadRequest, "not a btrfs subvolume")
			return
		}
		// move current to rollback
		curName := "current-" + ts
		curDst := filepath.Join(rbDir, curName)
		if _, err := exec.Command("btrfs", "subvolume", "snapshot", req.Path, curDst).CombinedOutput(); err != nil {
			logAuthPriv("rollback snapshot current failed")
		}
		// delete contents and restore from readonly snapshot clone
		snapSrc := filepath.Join(req.Path, ".snapshots", req.SnapshotID)
		// create rw snapshot at original path tmp and then move over
		tmpDst := filepath.Join(rbDir, "tmp-restore-"+ts)
		if _, err := exec.Command("btrfs", "subvolume", "snapshot", snapSrc, tmpDst).CombinedOutput(); err != nil {
			writeErr(w, http.StatusInternalServerError, "create rw snapshot failed")
			return
		}
		// replace original path: delete subvolume at original and snapshot back
		if _, err := exec.Command("btrfs", "subvolume", "delete", req.Path).CombinedOutput(); err != nil {
			writeErr(w, http.StatusInternalServerError, "delete current subvolume failed")
			return
		}
		if _, err := exec.Command("btrfs", "subvolume", "snapshot", tmpDst, req.Path).CombinedOutput(); err != nil {
			writeErr(w, http.StatusInternalServerError, "restore snapshot failed")
			return
		}
		_ = exec.Command("btrfs", "subvolume", "delete", tmpDst).Run()
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	case "tar":
		base := snapshotsTarDirForPath(req.Path)
		archive := filepath.Join(base, req.SnapshotID+".tar.gz")
		if _, err := os.Stat(archive); err != nil {
			writeErr(w, http.StatusBadRequest, "snapshot archive not found")
			return
		}
		// safety backup copy of current dir
		curBackup := filepath.Join(rbDir, "current-"+ts+".tar.gz")
		if _, err := exec.Command("tar", "-czf", curBackup, "-C", req.Path, ".").CombinedOutput(); err != nil {
			writeErr(w, http.StatusInternalServerError, "backup current failed")
			return
		}
		// extract over path
		if _, err := exec.Command("tar", "-xzf", archive, "-C", req.Path).CombinedOutput(); err != nil {
			writeErr(w, http.StatusInternalServerError, "extract failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	default:
		writeErr(w, http.StatusBadRequest, "invalid type")
		return
	}
}
