package pools

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
)

type Snapshot struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Readonly bool   `json:"readonly"`
}

// ListSnapshots returns subvolumes under the given mount as a best-effort list
func ListSnapshots(ctx context.Context, mount string) ([]Snapshot, error) {
	// btrfs subvolume list -o <mount>
	cmd := exec.CommandContext(ctx, "btrfs", "subvolume", "list", "-o", mount)
	out, err := cmd.Output()
	if err != nil {
		return []Snapshot{}, nil
	}
	snaps := []Snapshot{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		// line ends with path: e.g., "... path <path>"
		idx := strings.LastIndex(line, " path ")
		if idx >= 0 {
			p := strings.TrimSpace(line[idx+6:])
			name := p
			if i := strings.LastIndex(p, "/"); i >= 0 {
				name = p[i+1:]
			}
			snaps = append(snaps, Snapshot{Path: p, Name: name})
		}
	}
	return snaps, nil
}
