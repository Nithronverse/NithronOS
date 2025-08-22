package server

import (
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type BtrfsUsage struct {
	Total   uint64            `json:"total"`
	Used    uint64            `json:"used"`
	Free    uint64            `json:"free"`
	Classes map[string]RaidCl `json:"classes"`
}
type RaidCl struct {
	Profile string `json:"profile"`
	Total   uint64 `json:"total"`
	Used    uint64 `json:"used"`
}

func handleBtrfsUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	mount := r.URL.Query().Get("mount")
	if strings.TrimSpace(mount) == "" || !filepath.IsAbs(mount) {
		writeErr(w, http.StatusBadRequest, "absolute mount path required")
		return
	}
	out, err := exec.Command("btrfs", "filesystem", "usage", "--raw", mount).CombinedOutput()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, strings.TrimSpace(string(out)))
		return
	}
	u := parseBtrfsUsageRaw(string(out))
	writeJSON(w, http.StatusOK, u)
}

func parseBtrfsUsageRaw(s string) BtrfsUsage {
	u := BtrfsUsage{Classes: map[string]RaidCl{}}
	lines := strings.Split(s, "\n")
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		lower := strings.ToLower(t)
		if strings.HasPrefix(lower, "device size:") {
			u.Total = parseLastUint(t)
		} else if strings.HasPrefix(lower, "used:") {
			u.Used = parseLastUint(t)
		} else if strings.HasPrefix(lower, "free (estimated):") {
			u.Free = parseLastUint(t)
		} else if strings.HasPrefix(lower, "data,") || strings.HasPrefix(lower, "metadata,") || strings.HasPrefix(lower, "system,") {
			// e.g. "Data, RAID1: total=..., used=..."
			parts := strings.SplitN(t, ":", 2)
			head := parts[0]
			fields := ""
			if len(parts) > 1 {
				fields = parts[1]
			}
			segs := strings.Split(head, ",")
			kind := strings.ToLower(strings.TrimSpace(segs[0]))
			prof := ""
			if len(segs) > 1 {
				prof = strings.TrimSpace(segs[1])
			}
			rc := RaidCl{Profile: prof}
			for _, kv := range strings.Split(fields, ",") {
				kv = strings.TrimSpace(kv)
				if strings.HasPrefix(kv, "total=") {
					rc.Total = parseLastUint(kv)
				}
				if strings.HasPrefix(kv, "used=") {
					rc.Used = parseLastUint(kv)
				}
			}
			u.Classes[kind] = rc
		}
	}
	return u
}

func parseLastUint(s string) uint64 {
	// get trailing number sequence
	toks := strings.FieldsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
	if len(toks) == 0 {
		return 0
	}
	v, _ := strconv.ParseUint(toks[len(toks)-1], 10, 64)
	return v
}
