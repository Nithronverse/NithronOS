//go:build !windows

package server

import (
	"errors"
	"os"
	"syscall"
)

func mustBeRoot() error {
	if os.Geteuid() != 0 {
		return errors.New("nos-agent must run as root")
	}
	_ = syscall.Umask(0o002)
	return nil
}
