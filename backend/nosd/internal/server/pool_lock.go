package server

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	poolLockMu   sync.Mutex
	poolHeldByTx = map[string]string{} // poolId -> txId
)

func poolLockPath(id string) string {
	base := os.Getenv("NOS_STATE_DIR")
	if base == "" {
		base = "/var/lib/nos"
	}
	return filepath.Join(base, "locks", "pool."+sanitizeID(id)+".lock")
}

func sanitizeID(id string) string {
	out := make([]rune, 0, len(id))
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "unknown"
	}
	return string(out)
}

// tryAcquirePoolLock marks pool as busy with txId. Returns false if already held.
func tryAcquirePoolLock(poolID, txId string) bool {
	if os.Getenv("NOS_TEST_SKIP_POOL_LOCK") == "1" {
		return true
	}
	poolLockMu.Lock()
	defer poolLockMu.Unlock()
	if _, ok := poolHeldByTx[poolID]; ok {
		return false
	}
	// best-effort create marker file
	_ = os.MkdirAll(filepath.Dir(poolLockPath(poolID)), 0o755)
	f, _ := os.OpenFile(poolLockPath(poolID), os.O_CREATE|os.O_WRONLY, 0o644)
	if f != nil {
		_, _ = f.WriteString(txId)
		_ = f.Close()
	}
	poolHeldByTx[poolID] = txId
	return true
}

func releasePoolLock(poolID string) {
	if os.Getenv("NOS_TEST_SKIP_POOL_LOCK") == "1" {
		return
	}
	poolLockMu.Lock()
	defer poolLockMu.Unlock()
	delete(poolHeldByTx, poolID)
	_ = os.Remove(poolLockPath(poolID))
}

func currentPoolTx(poolID string) string {
	if os.Getenv("NOS_TEST_SKIP_POOL_LOCK") == "1" {
		return ""
	}
	poolLockMu.Lock()
	defer poolLockMu.Unlock()
	return poolHeldByTx[poolID]
}
