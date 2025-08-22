//go:build !prommetrics

package server

import "time"

func incBtrfsTx(action string)               {}
func observeBtrfsTxDuration(start time.Time) {}
func setBtrfsBalanceProgress(pct float64)    {}
func clearBtrfsBalanceProgress()             {}
