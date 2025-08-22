//go:build devdevice

package storage

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoopDeviceCreateSingle(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only")
	}
	if os.Geteuid() != 0 {
		t.Skip("requires root (EUID=0)")
	}
	if os.Getenv("NOS_DEVICE_TESTS") != "1" {
		t.Skip("set NOS_DEVICE_TESTS=1 to run")
	}
	if _, err := exec.LookPath("losetup"); err != nil {
		t.Skip("losetup not found")
	}
	if _, err := exec.LookPath("mkfs.btrfs"); err != nil {
		t.Skip("mkfs.btrfs not found")
	}

	// Sparse backing file
	tmpDir := t.TempDir()
	img := filepath.Join(tmpDir, "disk.img")
	f, err := os.Create(img)
	if err != nil {
		t.Fatalf("create img: %v", err)
	}
	if err := f.Truncate(1 << 30); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	_ = f.Close()

	// Attach loop device
	out, err := exec.Command("losetup", "--find", "--show", img).CombinedOutput()
	if err != nil {
		t.Fatalf("losetup attach: %v: %s", err, string(out))
	}
	loop := strings.TrimSpace(string(out))
	t.Logf("loop device: %s", loop)
	defer func() { _ = exec.Command("losetup", "-d", loop).Run() }()

	// Plan-like sequence
	// wipefs report (non-destructive)
	_ = exec.Command("wipefs", "-n", loop).Run()
	// mkfs
	if out, err = exec.Command("mkfs.btrfs", "-L", "testpool", "-d", "single", "-m", "single", loop).CombinedOutput(); err != nil {
		t.Fatalf("mkfs.btrfs: %v: %s", err, string(out))
	}

	// Mount
	mnt := filepath.Join(tmpDir, "mnt")
	if err := os.MkdirAll(mnt, 0o755); err != nil {
		t.Fatalf("mkdir mount: %v", err)
	}
	if out, err = exec.Command("mount", "-t", "btrfs", "-o", "noatime,compress=zstd:3", loop, mnt).CombinedOutput(); err != nil {
		t.Fatalf("mount: %v: %s", err, string(out))
	}
	defer func() { _ = exec.Command("umount", mnt).Run() }()

	// Create default subvolumes
	for _, sv := range []string{"data", "snaps", "apps"} {
		if out, err = exec.Command("btrfs", "subvolume", "create", filepath.Join(mnt, sv)).CombinedOutput(); err != nil {
			t.Fatalf("subvolume create %s: %v: %s", sv, err, string(out))
		}
		if _, statErr := os.Stat(filepath.Join(mnt, sv)); statErr != nil {
			t.Fatalf("missing subvol %s: %v", sv, statErr)
		}
	}

	// Assert mount is active by checking /proc/mounts
	mf, err := os.Open("/proc/mounts")
	if err != nil {
		t.Fatalf("open mounts: %v", err)
	}
	defer mf.Close()
	found := false
	scan := bufio.NewScanner(mf)
	for scan.Scan() {
		if strings.Contains(scan.Text(), " "+mnt+" ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mountpoint not found in /proc/mounts: %s", mnt)
	}
}
