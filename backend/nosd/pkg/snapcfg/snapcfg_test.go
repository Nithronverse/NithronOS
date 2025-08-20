package snapcfg

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

func TestLoad_SkipsMissingPaths(t *testing.T) {
	tmp := t.TempDir()
	cfgY := `version: 1
targets:
  - id: etc
    path: /does/not/exist
    type: auto
`
	p := writeTemp(t, tmp, "snapshots.yaml", cfgY)
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Targets) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(c.Targets))
	}
}

func TestLoad_ResolvesAutoType(t *testing.T) {
	tmp := t.TempDir()
	// create a real directory
	dir := filepath.Join(tmp, "apps")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgY := `version: 1
targets:
  - id: apps
    path: ` + dir + `
    type: auto
`
	p := writeTemp(t, tmp, "snapshots.yaml", cfgY)
	c, err := Load(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(c.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(c.Targets))
	}
	if c.Targets[0].DeclaredType != TargetTypeAuto {
		t.Fatalf("declared type not auto")
	}
	if c.Targets[0].Effective != TargetTypeTar && c.Targets[0].Effective != TargetTypeBtrfs {
		t.Fatalf("unexpected effective type: %s", c.Targets[0].Effective)
	}
}
