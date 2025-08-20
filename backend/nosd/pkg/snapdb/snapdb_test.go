package snapdb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withTempDB(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	_ = os.Setenv("NOS_SNAPDB_DIR", dir)
	return func() { _ = os.Unsetenv("NOS_SNAPDB_DIR") }
}

func TestEnsureDirAndAppendList(t *testing.T) {
	cleanup := withTempDB(t)
	defer cleanup()
	if err := EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	// index path should be under temp dir
	if _, err := os.Stat(filepath.Join(os.Getenv("NOS_SNAPDB_DIR"), "index.json")); !os.IsNotExist(err) {
		// it's okay if not exists yet; Append will create it
	}

	now := time.Now().UTC()
	tx := UpdateTx{
		TxID: "tx-1", StartedAt: now, Packages: []string{"nosd"}, Reason: "pre-update",
		Targets: []SnapshotTarget{{ID: "t1", Path: "/srv", Type: "tar", Location: "/var/lib/nos/snapshots/srv/abc.tar.gz", CreatedAt: now}},
	}
	if err := Append(tx); err != nil {
		t.Fatalf("Append: %v", err)
	}

	recent, err := ListRecent(10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(recent) != 1 || recent[0].TxID != "tx-1" {
		t.Fatalf("unexpected recent: %+v", recent)
	}

	got, err := FindByTx("tx-1")
	if err != nil {
		t.Fatalf("FindByTx: %v", err)
	}
	if got.TxID != "tx-1" || len(got.Targets) != 1 {
		t.Fatalf("unexpected tx: %+v", got)
	}
}
