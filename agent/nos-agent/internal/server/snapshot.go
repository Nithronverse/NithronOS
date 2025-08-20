package server

import (
	"context"
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

type SnapshotCreateRequest struct {
	TargetID     string   `json:"target_id"`
	Path         string   `json:"path"`
	Mode         string   `json:"mode"` // auto|btrfs|tar
	Reason       string   `json:"reason"`
	StopServices []string `json:"stop_services"`
}

type SnapshotCreateResponse struct {
	OK       bool   `json:"ok"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Location string `json:"location"`
}

func handleSnapshotCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runtime.GOOS == "windows" {
		writeErr(w, http.StatusNotImplemented, "not supported on windows")
		return
	}
	var req SnapshotCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Path == "" || !strings.HasPrefix(req.Path, "/") {
		writeErr(w, http.StatusBadRequest, "absolute path required")
		return
	}
	if req.Reason == "" {
		req.Reason = "manual"
	}
	fi, err := os.Stat(req.Path)
	if err != nil || !fi.IsDir() {
		writeErr(w, http.StatusBadRequest, "path must exist and be a directory")
		return
	}
	// stop services if requested
	stopped := []string{}
	if len(req.StopServices) > 0 {
		for _, s := range req.StopServices {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			cmd := exec.Command("systemctl", "stop", s)
			if out, err := cmd.CombinedOutput(); err != nil {
				// log but continue
				logAuthPriv(fmt.Sprintf("snapshot: stop %s failed: %s", s, string(out)))
			} else {
				stopped = append(stopped, s)
			}
		}
	}
	// always try to restart stopped services
	defer func() {
		for _, s := range stopped {
			cmd := exec.Command("systemctl", "start", s)
			if out, err := cmd.CombinedOutput(); err != nil {
				logAuthPriv(fmt.Sprintf("snapshot: start %s failed: %s", s, string(out)))
			}
		}
	}()

	tstamp := time.Now().UTC().Format("20060102-150405")
	reasonSlug := slugify(req.Reason)
	id := tstamp + "-" + reasonSlug

	mode := strings.ToLower(req.Mode)
	if mode == "" || mode == "auto" {
		// detect btrfs
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "btrfs", "subvolume", "show", req.Path)
		if out, err := cmd.CombinedOutput(); err == nil && len(out) > 0 {
			mode = "btrfs"
		} else {
			mode = "tar"
		}
	}

	switch mode {
	case "btrfs":
		snapDir := filepath.Join(req.Path, ".snapshots")
		if err := os.MkdirAll(snapDir, 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, "mkdir .snapshots failed")
			return
		}
		// try to mirror ownership of base path using chown on dir if we can get uid/gid via stat -c
		if out, err := exec.Command("/usr/bin/env", "sh", "-c", "stat -c %u:%g "+shellQuote(req.Path)).Output(); err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), ":")
			if len(parts) == 2 {
				_ = exec.Command("chown", parts[0]+":"+parts[1], snapDir).Run()
			}
		}
		dst := filepath.Join(snapDir, id)
		cmd := exec.Command("btrfs", "subvolume", "snapshot", "-r", req.Path, dst)
		if out, err := cmd.CombinedOutput(); err != nil {
			logAuthPriv(fmt.Sprintf("snapshot btrfs failed: %s", string(out)))
			writeErr(w, http.StatusInternalServerError, "btrfs snapshot failed")
			return
		}
		logAuthPriv(fmt.Sprintf("snapshot created type=btrfs path=%s dst=%s id=%s", req.Path, dst, id))
		writeJSON(w, http.StatusOK, SnapshotCreateResponse{OK: true, ID: id, Type: "btrfs", Location: dst})
		return
	case "tar":
		base := snapshotsTarDirForPath(req.Path)
		if err := os.MkdirAll(base, 0o755); err != nil {
			writeErr(w, http.StatusInternalServerError, "mkdir snapshots dir failed")
			return
		}
		dst := filepath.Join(base, id+".tar.gz")
		// try with xattrs+acls, fallback without
		tarArgs := []string{"-czf", dst, "-C", req.Path, "."}
		withEA := append([]string{"--xattrs", "--acls"}, tarArgs...)
		if out, err := exec.Command("tar", withEA...).CombinedOutput(); err != nil {
			if out2, err2 := exec.Command("tar", tarArgs...).CombinedOutput(); err2 != nil {
				logAuthPriv(fmt.Sprintf("snapshot tar failed: %s | %s", string(out), string(out2)))
				writeErr(w, http.StatusInternalServerError, "tar snapshot failed")
				return
			}
		}
		logAuthPriv(fmt.Sprintf("snapshot created type=tar path=%s dst=%s id=%s", req.Path, dst, id))
		writeJSON(w, http.StatusOK, SnapshotCreateResponse{OK: true, ID: id, Type: "tar", Location: dst})
		return
	default:
		writeErr(w, http.StatusBadRequest, "invalid mode")
		return
	}
}

func slugify(s string) string {
	s = strings.ToLower(s)
	// replace non-alnum with '-'
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out = append(out, r)
		} else {
			out = append(out, '-')
		}
	}
	res := strings.Trim(strings.ReplaceAll(string(out), "--", "-"), "-")
	if res == "" {
		res = "snap"
	}
	return res
}
