//go:build windows

package updates

import (
	"os"
)

// getAvailableDiskSpace returns available disk space in GB for the given path
// On Windows, we return a safe default since this is primarily for Linux systems
func getAvailableDiskSpace(path string) uint64 {
	// Return a large number to avoid false positives on Windows
	// The actual update system won't run on Windows anyway
	return 1000
}

// acquireFileLock is a stub for Windows
func acquireFileLock(file *os.File) error {
	// File locking not implemented on Windows for this system
	// This is OK since the update system is Linux-specific
	return nil
}

// releaseFileLock is a stub for Windows
func releaseFileLock(file *os.File) error {
	// File locking not implemented on Windows for this system
	return nil
}
