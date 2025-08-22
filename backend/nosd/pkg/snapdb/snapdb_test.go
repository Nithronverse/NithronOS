package snapdb

import (
	"encoding/json"
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
		// no-op; presence is fine, Append will create if missing
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

func TestAppendCreatesIndexAndIsReadable(t *testing.T) {
	cleanup := withTempDB(t)
	defer cleanup()
	dir := os.Getenv("NOS_SNAPDB_DIR")
	_ = os.Remove(filepath.Join(dir, "index.json"))
	now := time.Now().UTC()
	if err := Append(UpdateTx{TxID: "a1", StartedAt: now, Packages: []string{"nosd"}, Reason: "pre-update"}); err != nil {
		t.Fatalf("append: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	var items []UpdateTx
	if err := json.Unmarshal(b, &items); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(items) != 1 || items[0].TxID != "a1" {
		t.Fatalf("unexpected content: %s", string(b))
	}
	if _, err := os.Stat(filepath.Join(dir, "index.json.tmp")); err == nil {
		t.Fatalf("tmp file should not remain after atomic write")
	}
}

func TestListRecent_OrderAndLimit(t *testing.T) {
	cleanup := withTempDB(t)
	defer cleanup()
	base := time.Now().UTC()
	_ = Append(UpdateTx{TxID: "t1", StartedAt: base.Add(-3 * time.Hour)})
	_ = Append(UpdateTx{TxID: "t2", StartedAt: base.Add(-1 * time.Hour)})
	_ = Append(UpdateTx{TxID: "t3", StartedAt: base.Add(-2 * time.Hour)})
	got, err := ListRecent(2)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 || got[0].TxID != "t2" || got[1].TxID != "t3" {
		t.Fatalf("unexpected order/limit: %+v", got)
	}
	all, err := ListRecent(10)
	if err != nil || len(all) != 3 {
		t.Fatalf("want 3, got %d err=%v", len(all), err)
	}
}

func TestFindByTx_ReturnsRecord(t *testing.T) {
	cleanup := withTempDB(t)
	defer cleanup()
	now := time.Now().UTC()
	_ = Append(UpdateTx{TxID: "x1", StartedAt: now})
	_ = Append(UpdateTx{TxID: "x2", StartedAt: now.Add(1 * time.Minute)})
	rec, err := FindByTx("x2")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.TxID != "x2" {
		t.Fatalf("unexpected %+v", rec)
	}
}
