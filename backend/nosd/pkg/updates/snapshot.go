package updates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	snapshotBasePath = "/.snapshots/nos-update"
	rootSubvolume    = "@"
	etcSubvolume     = "@etc"
	varSubvolume     = "@var"
)

// SnapshotManager manages Btrfs snapshots for system updates
type SnapshotManager struct {
	basePath  string
	retention int // Number of snapshots to keep
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(retention int) *SnapshotManager {
	return &SnapshotManager{
		basePath:  snapshotBasePath,
		retention: retention,
	}
}

// IsBtrfs checks if the filesystem is Btrfs
func (sm *SnapshotManager) IsBtrfs(path string) (bool, error) {
	cmd := exec.Command("stat", "-f", "-c", "%T", path)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check filesystem type: %w", err)
	}

	fsType := strings.TrimSpace(string(output))
	return fsType == "btrfs", nil
}

// CreateSnapshot creates a snapshot of the system before an update
func (sm *SnapshotManager) CreateSnapshot(id string, description string) (*UpdateSnapshot, error) {
	// Check if filesystem is Btrfs
	isBtrfs, err := sm.IsBtrfs("/")
	if err != nil {
		return nil, err
	}
	if !isBtrfs {
		return nil, fmt.Errorf("filesystem is not Btrfs, snapshots not supported")
	}

	// Create snapshot directory
	snapshotDir := filepath.Join(sm.basePath, id)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Snapshot subvolumes
	subvolumes := []string{rootSubvolume, etcSubvolume, varSubvolume}
	for _, subvol := range subvolumes {
		if err := sm.snapshotSubvolume(subvol, snapshotDir); err != nil {
			// Clean up on failure
			_ = sm.DeleteSnapshot(id)
			return nil, fmt.Errorf("failed to snapshot %s: %w", subvol, err)
		}
	}

	// Get snapshot size
	size, _ := sm.getSnapshotSize(snapshotDir)

	// Create snapshot metadata
	snapshot := &UpdateSnapshot{
		ID:          id,
		CreatedAt:   time.Now(),
		Reason:      "update",
		Size:        size,
		Subvolumes:  subvolumes,
		CanRollback: true,
		Description: description,
	}

	return snapshot, nil
}

// snapshotSubvolume creates a snapshot of a single subvolume
func (sm *SnapshotManager) snapshotSubvolume(subvol, targetDir string) error {
	// Find the subvolume mount point
	mountPoint, err := sm.findSubvolumeMountPoint(subvol)
	if err != nil {
		return fmt.Errorf("failed to find mount point for %s: %w", subvol, err)
	}

	// Create read-only snapshot
	targetPath := filepath.Join(targetDir, subvol)
	cmd := exec.Command("btrfs", "subvolume", "snapshot", "-r", mountPoint, targetPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("btrfs snapshot failed: %s: %w", output, err)
	}

	return nil
}

// findSubvolumeMountPoint finds where a subvolume is mounted
func (sm *SnapshotManager) findSubvolumeMountPoint(subvol string) (string, error) {
	// Parse /proc/mounts to find the mount point
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 && strings.Contains(fields[3], "subvol="+subvol) {
			return fields[1], nil // Return mount point
		}
	}

	// Default mount points
	switch subvol {
	case rootSubvolume:
		return "/", nil
	case etcSubvolume:
		return "/etc", nil
	case varSubvolume:
		return "/var", nil
	default:
		return "", fmt.Errorf("unknown subvolume: %s", subvol)
	}
}

// ListSnapshots lists all update snapshots
func (sm *SnapshotManager) ListSnapshots() ([]UpdateSnapshot, error) {
	// Check if snapshot directory exists
	if _, err := os.Stat(sm.basePath); os.IsNotExist(err) {
		return []UpdateSnapshot{}, nil
	}

	entries, err := os.ReadDir(sm.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var snapshots []UpdateSnapshot
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		snapshotPath := filepath.Join(sm.basePath, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Get snapshot size
		size, _ := sm.getSnapshotSize(snapshotPath)

		// Check which subvolumes exist
		var subvolumes []string
		for _, subvol := range []string{rootSubvolume, etcSubvolume, varSubvolume} {
			if _, err := os.Stat(filepath.Join(snapshotPath, subvol)); err == nil {
				subvolumes = append(subvolumes, subvol)
			}
		}

		snapshot := UpdateSnapshot{
			ID:          entry.Name(),
			CreatedAt:   info.ModTime(),
			Size:        size,
			Subvolumes:  subvolumes,
			CanRollback: true,
			Reason:      "update",
		}

		snapshots = append(snapshots, snapshot)
	}

	// Sort by creation time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

// DeleteSnapshot deletes a snapshot
func (sm *SnapshotManager) DeleteSnapshot(id string) error {
	snapshotPath := filepath.Join(sm.basePath, id)

	// Check if snapshot exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", id)
	}

	// Delete each subvolume snapshot
	for _, subvol := range []string{rootSubvolume, etcSubvolume, varSubvolume} {
		subvolPath := filepath.Join(snapshotPath, subvol)
		if _, err := os.Stat(subvolPath); err == nil {
			cmd := exec.Command("btrfs", "subvolume", "delete", subvolPath)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to delete subvolume %s: %s: %w", subvol, output, err)
			}
		}
	}

	// Remove the snapshot directory
	if err := os.RemoveAll(snapshotPath); err != nil {
		return fmt.Errorf("failed to remove snapshot directory: %w", err)
	}

	return nil
}

// Rollback rolls back to a previous snapshot
func (sm *SnapshotManager) Rollback(snapshotID string) error {
	snapshotPath := filepath.Join(sm.basePath, snapshotID)

	// Verify snapshot exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", snapshotID)
	}

	// Create a recovery snapshot before rollback
	recoveryID := fmt.Sprintf("recovery-%d", time.Now().Unix())
	if _, err := sm.CreateSnapshot(recoveryID, "Pre-rollback recovery snapshot"); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to create recovery snapshot: %v\n", err)
	}

	// Rollback each subvolume
	for _, subvol := range []string{rootSubvolume, etcSubvolume, varSubvolume} {
		if err := sm.rollbackSubvolume(subvol, snapshotPath); err != nil {
			return fmt.Errorf("failed to rollback %s: %w", subvol, err)
		}
	}

	// Regenerate initramfs if needed
	if err := sm.regenerateInitramfs(); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to regenerate initramfs: %v\n", err)
	}

	return nil
}

// rollbackSubvolume rolls back a single subvolume
func (sm *SnapshotManager) rollbackSubvolume(subvol, snapshotPath string) error {
	snapshotSubvolPath := filepath.Join(snapshotPath, subvol)

	// Check if snapshot subvolume exists
	if _, err := os.Stat(snapshotSubvolPath); os.IsNotExist(err) {
		// Subvolume might not have been snapshotted
		return nil
	}

	// Find the current mount point
	mountPoint, err := sm.findSubvolumeMountPoint(subvol)
	if err != nil {
		return err
	}

	// Create a temporary backup of current state
	tempBackup := mountPoint + ".rollback-backup"
	cmd := exec.Command("btrfs", "subvolume", "snapshot", mountPoint, tempBackup)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create backup: %s: %w", output, err)
	}

	// Delete the current subvolume
	cmd = exec.Command("btrfs", "subvolume", "delete", mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try to restore from temp backup
		if mvErr := exec.Command("mv", tempBackup, mountPoint).Run(); mvErr != nil {
			fmt.Printf("Failed to restore backup: %v\n", mvErr)
		}
		return fmt.Errorf("failed to delete current subvolume: %s: %w", output, err)
	}

	// Create new snapshot from the saved snapshot (read-write this time)
	cmd = exec.Command("btrfs", "subvolume", "snapshot", snapshotSubvolPath, mountPoint)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try to restore from temp backup
		if mvErr := exec.Command("mv", tempBackup, mountPoint).Run(); mvErr != nil {
			fmt.Printf("Failed to restore backup: %v\n", mvErr)
		}
		return fmt.Errorf("failed to restore snapshot: %s: %w", output, err)
	}

	// Clean up temp backup
	if err := exec.Command("btrfs", "subvolume", "delete", tempBackup).Run(); err != nil {
		fmt.Printf("Failed to delete temp backup: %v\n", err)
	}

	return nil
}

// PruneSnapshots removes old snapshots beyond retention limit
func (sm *SnapshotManager) PruneSnapshots() error {
	snapshots, err := sm.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Keep only the configured number of snapshots
	if len(snapshots) > sm.retention {
		// Snapshots are already sorted by creation time (newest first)
		for i := sm.retention; i < len(snapshots); i++ {
			if err := sm.DeleteSnapshot(snapshots[i].ID); err != nil {
				// Log but continue
				fmt.Printf("Warning: failed to delete old snapshot %s: %v\n", snapshots[i].ID, err)
			}
		}
	}

	return nil
}

// getSnapshotSize calculates the size of a snapshot
func (sm *SnapshotManager) getSnapshotSize(path string) (int64, error) {
	cmd := exec.Command("du", "-sb", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(output))
	if len(fields) > 0 {
		size, _ := strconv.ParseInt(fields[0], 10, 64)
		return size, nil
	}

	return 0, nil
}

// regenerateInitramfs regenerates the initramfs after rollback
func (sm *SnapshotManager) regenerateInitramfs() error {
	// Try update-initramfs (Debian/Ubuntu)
	if _, err := exec.LookPath("update-initramfs"); err == nil {
		cmd := exec.Command("update-initramfs", "-u", "-k", "all")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("update-initramfs failed: %s: %w", output, err)
		}
		return nil
	}

	// Try dracut (alternative)
	if _, err := exec.LookPath("dracut"); err == nil {
		cmd := exec.Command("dracut", "-f")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("dracut failed: %s: %w", output, err)
		}
		return nil
	}

	return fmt.Errorf("no initramfs tool found")
}

// VerifySnapshot verifies that a snapshot is valid and complete
func (sm *SnapshotManager) VerifySnapshot(id string) error {
	snapshotPath := filepath.Join(sm.basePath, id)

	// Check if snapshot directory exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s not found", id)
	}

	// Verify each subvolume
	for _, subvol := range []string{rootSubvolume, etcSubvolume, varSubvolume} {
		subvolPath := filepath.Join(snapshotPath, subvol)

		// Check if subvolume exists
		if _, err := os.Stat(subvolPath); os.IsNotExist(err) {
			continue // Some subvolumes might be optional
		}

		// Verify it's a Btrfs subvolume
		cmd := exec.Command("btrfs", "subvolume", "show", subvolPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("invalid subvolume %s: %s: %w", subvol, output, err)
		}
	}

	return nil
}
