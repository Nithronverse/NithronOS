package ratelimit

import (
	"path/filepath"
	"testing"
)

func TestRateLimitRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ratelimit.json")
	s := New(path)
	if err := s.Put("ip:1.2.3.4", Bucket{Hits: 3, Window: NowUTC()}); err != nil {
		t.Fatalf("put: %v", err)
	}
	st := s.Snapshot()
	if st.Buckets["ip:1.2.3.4"].Hits != 3 {
		t.Fatalf("bad hits: %+v", st)
	}
}
