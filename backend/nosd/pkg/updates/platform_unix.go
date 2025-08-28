//go:build linux || darwin || freebsd

package updates

import (
	"os"
	"syscall"
)

// getAvailableDiskSpace returns available disk space in GB for the given path
func getAvailableDiskSpace(path string) uint64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		// Return a large number to avoid false positives
		return 1000
	}
	return (stat.Bavail * uint64(stat.Bsize)) / (1024 * 1024 * 1024)
}

// acquireFileLock acquires an exclusive lock on a file
func acquireFileLock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// releaseFileLock releases a file lock
func releaseFileLock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
