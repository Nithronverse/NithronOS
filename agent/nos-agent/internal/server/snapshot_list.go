package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type SnapshotListRequest struct {
	Path string `json:"path"`
}

type SnapshotEntry struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
	SizeBytes int64  `json:"size"`
	Location  string `json:"location"`
}

type SnapshotListResponse struct {
	Items []SnapshotEntry `json:"items"`
}

func handleSnapshotList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runtime.GOOS == "windows" {
		writeErr(w, http.StatusNotImplemented, "not supported on windows")
		return
	}
	var req SnapshotListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Path == "" || !strings.HasPrefix(req.Path, "/") {
		writeErr(w, http.StatusBadRequest, "absolute path required")
		return
	}
	// btrfs snapshots
	items := make([]SnapshotEntry, 0, 8)
	snapDir := filepath.Join(req.Path, ".snapshots")
	if ents, err := os.ReadDir(snapDir); err == nil {
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(snapDir, e.Name())
			info, _ := os.Stat(p)
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			items = append(items, SnapshotEntry{
				Type: "btrfs", Name: e.Name(), Timestamp: parseTimestampFromName(e.Name()), SizeBytes: size, Location: p,
			})
		}
	}
	// tar snapshots
	tarBase := snapshotsTarDirForPath(req.Path)
	if ents, err := os.ReadDir(tarBase); err == nil {
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			if !strings.HasSuffix(e.Name(), ".tar.gz") {
				continue
			}
			p := filepath.Join(tarBase, e.Name())
			info, _ := os.Stat(p)
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			items = append(items, SnapshotEntry{
				Type: "tar", Name: strings.TrimSuffix(e.Name(), ".tar.gz"), Timestamp: parseTimestampFromName(e.Name()), SizeBytes: size, Location: p,
			})
		}
	}
	writeJSON(w, http.StatusOK, SnapshotListResponse{Items: items})
}

func parseTimestampFromName(name string) string {
	// expects leading yyyyMMdd-HHmmss
	name = strings.TrimSuffix(name, ".tar.gz")
	if len(name) >= 15 && name[8] == '-' {
		return name[:15]
	}
	return ""
}
