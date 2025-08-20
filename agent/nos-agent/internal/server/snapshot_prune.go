package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type SnapshotPruneRequest struct {
	KeepPerTarget int      `json:"keep_per_target"`
	Paths         []string `json:"paths"`
}

func handleSnapshotPrune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if runtime.GOOS == "windows" {
		writeErr(w, http.StatusNotImplemented, "not supported on windows")
		return
	}
	var req SnapshotPruneRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.KeepPerTarget <= 0 {
		req.KeepPerTarget = 5
	}

	candidates := []string{}
	if len(req.Paths) > 0 {
		candidates = append(candidates, req.Paths...)
	} else {
		// try common roots for btrfs snapshots
		for _, root := range []string{"/srv", "/mnt"} {
			_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() && filepath.Base(p) == ".snapshots" {
					candidates = append(candidates, filepath.Dir(p))
					return filepath.SkipDir
				}
				// limit depth
				if strings.Count(strings.TrimPrefix(p, root), string(filepath.Separator)) > 2 {
					return filepath.SkipDir
				}
				return nil
			})
		}
		// add tar snapshot dirs under snapshot base
		base := snapshotsBaseDir()
		if ents, err := os.ReadDir(base); err == nil {
			for _, e := range ents {
				if e.IsDir() {
					candidates = append(candidates, filepath.Join(base, e.Name()))
				}
			}
		}
	}

	pruned := map[string]int{}
	for _, c := range candidates {
		// If path is a base dir that has .snapshots (btrfs)
		snapDir := filepath.Join(c, ".snapshots")
		if fi, err := os.Stat(snapDir); err == nil && fi.IsDir() {
			n := pruneDirs(snapDir, req.KeepPerTarget, true)
			pruned[c+" (btrfs)"] = n
			continue
		}
		// If path is a tar dir (under snapshotsBaseDir)
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			n := pruneFiles(c, req.KeepPerTarget)
			pruned[c+" (tar)"] = n
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "pruned": pruned})
}

// pruneDirs keeps newest N directories by modtime, deletes the rest.
// If btrfs is true, use `btrfs subvolume delete` to remove.
func pruneDirs(dir string, keep int, btrfs bool) int {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	type item struct {
		name string
		mod  int64
	}
	items := []item{}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{name: e.Name(), mod: info.ModTime().Unix()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod > items[j].mod })
	del := 0
	for i := keep; i < len(items); i++ {
		path := filepath.Join(dir, items[i].name)
		if btrfs {
			_ = exec.Command("btrfs", "subvolume", "delete", path).Run()
		} else {
			_ = os.RemoveAll(path)
		}
		del++
	}
	return del
}

// pruneFiles keeps newest N *.tar.gz files by modtime.
func pruneFiles(dir string, keep int) int {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	type item struct {
		name string
		mod  int64
	}
	items := []item{}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, item{name: e.Name(), mod: info.ModTime().Unix()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod > items[j].mod })
	del := 0
	for i := keep; i < len(items); i++ {
		path := filepath.Join(dir, items[i].name)
		_ = os.Remove(path)
		del++
	}
	return del
}
