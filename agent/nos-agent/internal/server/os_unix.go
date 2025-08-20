//go:build !windows

package server

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"syscall"
)

func mustBeRoot() error {
	if os.Geteuid() != 0 {
		return errors.New("nos-agent must run as root")
	}
	_ = syscall.Umask(0o002)
	return nil
}

func runtimeChownSupported() bool { return true }

func chownByName(path, owner, group string) error {
	var uid, gid int
	if owner != "" {
		if u, err := user.Lookup(owner); err == nil {
			if id, err2 := atoi(u.Uid); err2 == nil {
				uid = id
			}
		}
	}
	if group != "" {
		if g, err := user.LookupGroup(group); err == nil {
			if id, err2 := atoi(g.Gid); err2 == nil {
				gid = id
			}
		}
	}
	if uid == 0 && gid == 0 {
		return nil
	}
	return syscall.Chown(path, uid, gid)
}

func atoi(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
