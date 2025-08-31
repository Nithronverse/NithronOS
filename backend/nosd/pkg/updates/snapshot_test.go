package updates

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSnapshotManager(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping snapshot manager tests on non-Linux")
	}
	// Skip if not running on Btrfs
	sm := NewSnapshotManager(3)
	isBtrfs, _ := sm.IsBtrfs("/tmp")
	if !isBtrfs {
		t.Skip("Skipping Btrfs-specific tests")
	}

	t.Run("CreateSnapshot", func(t *testing.T) {
		id := "test-snapshot-" + time.Now().Format("20060102150405")
		snapshot, err := sm.CreateSnapshot(id, "Test snapshot")
		if err != nil {
			// May fail if not root or Btrfs not available
			t.Skipf("Cannot create snapshot: %v", err)
		}

		if snapshot.ID != id {
			t.Errorf("Expected snapshot ID %s, got %s", id, snapshot.ID)
		}

		// Clean up
		_ = sm.DeleteSnapshot(id)
	})

	t.Run("ListSnapshots", func(t *testing.T) {
		snapshots, err := sm.ListSnapshots()
		if err != nil {
			t.Fatalf("Failed to list snapshots: %v", err)
		}

		// Should return empty list if no snapshots
		if snapshots == nil {
			t.Error("Expected snapshots list, got nil")
		}
	})

	t.Run("PruneSnapshots", func(t *testing.T) {
		// Create multiple snapshots
		for i := 0; i < 5; i++ {
			id := fmt.Sprintf("prune-test-%d-%d", i, time.Now().Unix())
			_, err := sm.CreateSnapshot(id, "Prune test")
			if err != nil {
				t.Skipf("Cannot create test snapshots: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Prune to keep only 3
		err := sm.PruneSnapshots()
		if err != nil {
			t.Errorf("Failed to prune snapshots: %v", err)
		}

		snapshots, _ := sm.ListSnapshots()
		if len(snapshots) > sm.retention {
			t.Errorf("Expected at most %d snapshots after pruning, got %d",
				sm.retention, len(snapshots))
		}

		// Clean up remaining
		for _, s := range snapshots {
			if strings.HasPrefix(s.ID, "prune-test-") {
				_ = sm.DeleteSnapshot(s.ID)
			}
		}
	})
}

func TestSnapshotValidation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping snapshot validation tests on non-Linux")
	}
	sm := NewSnapshotManager(3)

	t.Run("VerifyNonExistentSnapshot", func(t *testing.T) {
		err := sm.VerifySnapshot("non-existent-snapshot")
		if err == nil {
			t.Error("Expected error for non-existent snapshot")
		}
	})

	t.Run("IsBtrfs", func(t *testing.T) {
		// Test with /tmp which is usually not Btrfs
		isBtrfs, err := sm.IsBtrfs("/tmp")
		if err != nil {
			t.Errorf("Failed to check filesystem type: %v", err)
		}

		// Result depends on actual filesystem
		t.Logf("/tmp is Btrfs: %v", isBtrfs)
	})
}

func TestSnapshotHelpers(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping snapshot helper tests on non-Linux")
	}
	sm := NewSnapshotManager(3)

	t.Run("FindSubvolumeMountPoint", func(t *testing.T) {
		// Test with known subvolumes
		mountPoint, err := sm.findSubvolumeMountPoint(rootSubvolume)
		if err != nil {
			t.Errorf("Failed to find root subvolume mount point: %v", err)
		}

		if mountPoint != "/" {
			t.Errorf("Expected root mount point /, got %s", mountPoint)
		}
	})

	t.Run("GetSnapshotSize", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "snapshot-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Write test file
		testFile := filepath.Join(tmpDir, "test.txt")
		_ = os.WriteFile(testFile, []byte("test content"), 0644)

		size, err := sm.getSnapshotSize(tmpDir)
		if err != nil {
			t.Errorf("Failed to get snapshot size: %v", err)
		}

		if size <= 0 {
			t.Error("Expected positive size")
		}
	})
}
