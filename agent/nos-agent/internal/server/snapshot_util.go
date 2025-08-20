package server

import (
	"fmt"
	"path/filepath"
	"time"
)

// buildBtrfsSnapshotDst returns the destination path for a read-only btrfs snapshot
// created under <base>/.snapshots/<timestamp>-<reasonSlug>.
func buildBtrfsSnapshotDst(base string, ts time.Time, reason string) string {
	id := ts.UTC().Format("20060102-150405") + "-" + slugify(reason)
	return filepath.Join(base, ".snapshots", id)
}

// buildTarSnapshotPath returns the destination file path for a tar snapshot
// stored under snapshotsTarDirForPath(target), named <timestamp>-<reasonSlug>.tar.gz
func buildTarSnapshotPath(target string, ts time.Time, reason string) string {
	base := snapshotsTarDirForPath(target)
	id := ts.UTC().Format("20060102-150405") + "-" + slugify(reason)
	return filepath.Join(base, fmt.Sprintf("%s.tar.gz", id))
}

// pruneSelection returns the names of entries to delete when keeping the newest keepN entries.
// The input slice must be provided in any order; the result is ordered oldest→newest among deletions.
type PruneEntry struct {
	Name    string
	ModTime time.Time
}

func pruneSelection(entries []PruneEntry, keepN int) []string {
	if keepN < 0 {
		keepN = 0
	}
	// simple insertion sort by ModTime desc (newest first) for small N
	e := make([]PruneEntry, len(entries))
	copy(e, entries)
	for i := 1; i < len(e); i++ {
		j := i
		for j > 0 && e[j-1].ModTime.Before(e[j].ModTime) {
			e[j-1], e[j] = e[j], e[j-1]
			j--
		}
	}
	if keepN >= len(e) {
		return []string{}
	}
	// items to delete are those after keepN, returned oldest→newest
	dels := e[keepN:]
	// reverse to oldest→newest (currently newest-first)
	for i, j := 0, len(dels)-1; i < j; i, j = i+1, j-1 {
		dels[i], dels[j] = dels[j], dels[i]
	}
	out := make([]string, len(dels))
	for i, d := range dels {
		out[i] = d.Name
	}
	return out
}
