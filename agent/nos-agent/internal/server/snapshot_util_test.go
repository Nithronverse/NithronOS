package server

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestBuildBtrfsSnapshotDst(t *testing.T) {
	ts := time.Date(2025, 8, 20, 12, 34, 56, 0, time.UTC)
	p := buildBtrfsSnapshotDst("/srv/data", ts, "pre-update")
	want := filepath.Join("/srv/data", ".snapshots", "20250820-123456-pre-update")
	if p != want {
		t.Fatalf("got %s want %s", p, want)
	}
}

func TestBuildTarSnapshotPath(t *testing.T) {
	ts := time.Date(2025, 8, 20, 12, 34, 56, 0, time.UTC)
	p := buildTarSnapshotPath("/etc/nos", ts, "pre-update")
	base := snapshotsTarDirForPath("/etc/nos")
	want := filepath.Join(base, "20250820-123456-pre-update.tar.gz")
	if p != want {
		t.Fatalf("got %s want %s", p, want)
	}
}

func TestPruneSelection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path semantics differ but logic is OS-agnostic")
	}
	base := time.Date(2025, 8, 20, 12, 0, 0, 0, time.UTC)
	entries := []PruneEntry{
		{Name: "a", ModTime: base.Add(10 * time.Minute)}, // newest
		{Name: "b", ModTime: base.Add(5 * time.Minute)},
		{Name: "c", ModTime: base.Add(1 * time.Minute)},
		{Name: "d", ModTime: base.Add(2 * time.Minute)},
	}
	dels := pruneSelection(entries, 2)
	// keep two newest: a,b → delete c,d in oldest→newest: c then d
	if len(dels) != 2 || dels[0] != "c" || dels[1] != "d" {
		t.Fatalf("unexpected dels: %+v", dels)
	}
}
