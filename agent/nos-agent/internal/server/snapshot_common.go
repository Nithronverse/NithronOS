package server

import (
	"os"
	"path/filepath"
)

func snapshotsBaseDir() string {
	if v := os.Getenv("NOS_SNAPSHOT_BASE"); v != "" {
		return v
	}
	return "/var/lib/nos/snapshots"
}

func snapshotsTarDirForPath(path string) string {
	return filepath.Join(snapshotsBaseDir(), slugify(path))
}
