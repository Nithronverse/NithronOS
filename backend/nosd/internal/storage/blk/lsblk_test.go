package blk

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNormalizeSize(t *testing.T) {
	if got := normalizeSize(json.Number("8589934592")); got != 8589934592 {
		t.Fatalf("expected 8GiB, got %d", got)
	}
}

func TestParseFixture(t *testing.T) {
	b, err := os.ReadFile("../../server/testdata/lsblk.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var tree rawTree
	if err := json.Unmarshal(b, &tree); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ds := flatten(tree)
	if len(ds) == 0 {
		t.Fatalf("no devices")
	}
}
