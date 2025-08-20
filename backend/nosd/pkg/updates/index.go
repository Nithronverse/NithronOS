package updates

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type SnapshotRef struct {
	TargetID   string `json:"target_id"`
	TargetPath string `json:"target_path"`
	Type       string `json:"type"` // btrfs|tar
	SnapshotID string `json:"snapshot_id"`
	Location   string `json:"location"`
}

type Transaction struct {
	TxID     string        `json:"tx_id"`
	Time     time.Time     `json:"time"`
	Targets  []SnapshotRef `json:"targets"`
	Packages []string      `json:"packages"`
	Result   string        `json:"result"`
}

type Index struct {
	Transactions []Transaction `json:"transactions"`
}

func defaultIndexPath() string {
	// Dev default if present
	dev := filepath.Clean("./devdata/snapshots-index.json")
	if _, err := os.Stat(dev); err == nil {
		return dev
	}
	return "/var/lib/nos/snapshots/index.json"
}

func Load(path string) (Index, error) {
	if path == "" {
		path = defaultIndexPath()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Index{}, nil
		}
		return Index{}, err
	}
	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return Index{}, err
	}
	return idx, nil
}

func Save(path string, idx Index) error {
	if path == "" {
		path = defaultIndexPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
