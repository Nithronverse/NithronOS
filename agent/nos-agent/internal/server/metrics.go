package server

import (
	"sync"
)

// naive in-memory counters and histograms (per-process)

var (
	btrfsStatusMu             sync.Mutex
	btrfsStatusCallsTotal     = map[string]uint64{}             // kind -> count
	btrfsStatusLatencyBuckets = map[string]map[float64]uint64{} // kind -> upperBoundSec -> count
	btrfsStatusLatencySum     = map[string]float64{}            // kind -> seconds sum
	btrfsStatusLatencyCount   = map[string]uint64{}             // kind -> samples
	btrfsStatusLatencyBounds  = []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10}
)

func recordBtrfsStatus(kind string, seconds float64) {
	btrfsStatusMu.Lock()
	defer btrfsStatusMu.Unlock()
	btrfsStatusCallsTotal[kind]++
	btrfsStatusLatencySum[kind] += seconds
	btrfsStatusLatencyCount[kind]++
	bk, ok := btrfsStatusLatencyBuckets[kind]
	if !ok {
		bk = map[float64]uint64{}
		btrfsStatusLatencyBuckets[kind] = bk
	}
	for _, ub := range btrfsStatusLatencyBounds {
		if seconds <= ub {
			bk[ub]++
			return
		}
	}
	// overflow bucket: +Inf
	bk[1e9]++

	// If Prometheus is built in (prommetrics tag), update prom collectors too.
	incBtrfsStatus(kind)
	observeBtrfsStatus(kind, seconds)
}
