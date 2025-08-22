package server

import (
	"fmt"
	"regexp"
	"strings"
)

// BalanceInfo represents parsed details from `btrfs balance status`.
type BalanceInfo struct {
	Running bool
	Percent float64
	Left    string
	Total   string
}

// balanceStatus parses `btrfs balance status <mount>` output.
// Returns running flag and best-effort progress percent.
func balanceStatus(out string) (running bool, pct float64) {
	s := strings.ToLower(out)
	if strings.Contains(s, "no balance found") || strings.Contains(s, "not running") {
		return false, 0
	}
	running = strings.Contains(s, "running") || strings.Contains(s, "progress")
	// Match e.g. "(0/1) 17%", "completed: 42 %"
	re := regexp.MustCompile(`(?i)(\d+\.?\d*)\s*%`)
	if m := re.FindStringSubmatch(out); len(m) == 2 {
		// parse percent
		var v float64
		_, _ = fmt.Sscanf(m[1], "%f", &v)
		pct = v
	}
	return running, pct
}

// parseBalanceInfo parses out running, percent, and best-effort left/total chunk counts.
func parseBalanceInfo(out string) BalanceInfo {
	run, pct := balanceStatus(out)
	info := BalanceInfo{Running: run, Percent: pct}
	// Try to parse "<done> out of about <total> chunks balanced"
	// Examples:
	//   "  123 out of about 1000 chunks balanced (123 considered),  87% left"
	//   "  1000 out of 1000 chunks balanced"
	re := regexp.MustCompile(`(?i)\b(\d+)\s+out of\s+(?:about\s+)?(\d+)\s+chunks\s+balanced`)
	if m := re.FindStringSubmatch(out); len(m) == 3 {
		var done, total int
		_, _ = fmt.Sscanf(m[1], "%d", &done)
		_, _ = fmt.Sscanf(m[2], "%d", &total)
		if total > 0 {
			info.Total = fmt.Sprintf("%d", total)
			if total-done >= 0 {
				info.Left = fmt.Sprintf("%d", total-done)
			}
		}
	}
	return info
}

// replaceStatus is best-effort; accepts `btrfs replace status` output
func replaceStatus(out string) (state string) {
	s := strings.ToLower(out)
	switch {
	case strings.Contains(s, "finished"):
		return "finished"
	case strings.Contains(s, "running"):
		return "running"
	case strings.Contains(s, "never started"):
		return "idle"
	default:
		return "unknown"
	}
}

// ReplaceInfo represents parsed details from `btrfs replace status`.
type ReplaceInfo struct {
	Running   bool
	Completed int
	Total     int
	Percent   float64
}

// parseReplaceInfo parses `btrfs replace status` for running flag, completed/total items and percent.
func parseReplaceInfo(out string) ReplaceInfo {
	s := strings.ToLower(out)
	info := ReplaceInfo{}
	if strings.Contains(s, "finished") {
		info.Running = false
	} else if strings.Contains(s, "never started") || strings.Contains(s, "not started") {
		info.Running = false
	} else if strings.Contains(s, "running") || strings.Contains(s, "started") {
		info.Running = true
	}
	// percent
	rePct := regexp.MustCompile(`(?i)(\d+\.?\d*)\s*%`)
	if m := rePct.FindStringSubmatch(out); len(m) == 2 {
		var v float64
		_, _ = fmt.Sscanf(m[1], "%f", &v)
		info.Percent = v
	}
	// completed/total like "3/30" or "123 / 456"
	reCT := regexp.MustCompile(`(?i)\b(\d+)\s*/\s*(\d+)\b`)
	if m := reCT.FindStringSubmatch(out); len(m) == 3 {
		_, _ = fmt.Sscanf(m[1], "%d", &info.Completed)
		_, _ = fmt.Sscanf(m[2], "%d", &info.Total)
	}
	return info
}
