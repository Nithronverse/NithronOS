//go:build windows

package fsatomic

import (
	"errors"
	"os"
	"time"
)

// flockExclusive on Windows approximates an exclusive advisory lock using
// create-excl of the lock file. It retries until it can create the lock file
// and removes it on unlock.
func flockExclusive(lockPath string) (func(), error) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
		if err == nil {
			// Hold f open and remove lock file on unlock
			unlocked := false
			return func() {
				if unlocked {
					return
				}
				_ = f.Close()
				_ = os.Remove(lockPath)
				unlocked = true
			}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, errors.New("lock timeout")
		}
		time.Sleep(25 * time.Millisecond)
	}
}
