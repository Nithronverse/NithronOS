//go:build prommetrics

package server

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	btrfsTxTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "btrfs_tx_total",
			Help: "Total number of btrfs device transactions by action.",
		},
		[]string{"action"},
	)
	btrfsTxDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "btrfs_tx_duration_seconds",
			Help:    "Duration of btrfs device transactions in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
	btrfsBalanceProgressGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "btrfs_balance_progress_percent",
			Help: "Current balance progress percent (temporary).",
		},
	)
)

func init() {
	prometheus.MustRegister(btrfsTxTotal)
	prometheus.MustRegister(btrfsTxDuration)
	prometheus.MustRegister(btrfsBalanceProgressGauge)
}

func incBtrfsTx(action string)               { btrfsTxTotal.WithLabelValues(action).Inc() }
func observeBtrfsTxDuration(start time.Time) { btrfsTxDuration.Observe(time.Since(start).Seconds()) }
func setBtrfsBalanceProgress(pct float64)    { btrfsBalanceProgressGauge.Set(pct) }
func clearBtrfsBalanceProgress()             { btrfsBalanceProgressGauge.Set(0) }
