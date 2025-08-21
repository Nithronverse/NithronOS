//go:build !windows

package fsatomic

import (
	"os"

	"golang.org/x/sys/unix"
)

func flockExclusive(lockPath string) (func(), error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	unlocked := false
	return func() {
		if unlocked {
			return
		}
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		_ = f.Close()
		unlocked = true
	}, nil
}
