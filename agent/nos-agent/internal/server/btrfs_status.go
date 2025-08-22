package server

import (
	"bytes"
	"context"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

func handleBtrfsBalanceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	mount := r.URL.Query().Get("mount")
	if !filepath.IsAbs(mount) {
		writeErr(w, http.StatusBadRequest, "mount required")
		return
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/btrfs", "balance", "status", mount)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	out := stdout.String()
	if stderr.Len() > 0 {
		out += stderr.String()
	}
	info := parseBalanceInfo(out)
	recordBtrfsStatus("balance", time.Since(start).Seconds())
	resp := map[string]any{"running": info.Running, "raw": out}
	if info.Percent > 0 {
		resp["percent"] = info.Percent
	}
	if info.Left != "" {
		resp["left"] = info.Left
	}
	if info.Total != "" {
		resp["total"] = info.Total
	}
	logAuthPriv("agent.status.balance mount=" + mount + " running=" + boolToStr(info.Running) + optPct(info.Percent))
	writeJSON(w, http.StatusOK, resp)
}

func handleBtrfsReplaceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	mount := r.URL.Query().Get("mount")
	if !filepath.IsAbs(mount) {
		writeErr(w, http.StatusBadRequest, "mount required")
		return
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/btrfs", "replace", "status", mount)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	out := stdout.String()
	if stderr.Len() > 0 {
		out += stderr.String()
	}
	ri := parseReplaceInfo(out)
	recordBtrfsStatus("replace", time.Since(start).Seconds())
	resp := map[string]any{"running": ri.Running, "raw": out}
	if ri.Percent > 0 {
		resp["percent"] = ri.Percent
	}
	if ri.Completed > 0 {
		resp["completed"] = ri.Completed
	}
	if ri.Total > 0 {
		resp["total"] = ri.Total
	}
	logAuthPriv("agent.status.replace mount=" + mount + " running=" + boolToStr(ri.Running) + optPct(ri.Percent))
	writeJSON(w, http.StatusOK, resp)
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func optPct(p float64) string {
	if p <= 0 {
		return ""
	}
	return " percent=" + strconv.FormatFloat(p, 'f', 1, 64)
}
