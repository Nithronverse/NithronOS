package server

import "testing"

func TestBalanceStatusParsesPercent(t *testing.T) {
	out := "Balance on '/mnt/p': running\n  12% done, 0 errors\n"
	run, pct := balanceStatus(out)
	if !run {
		t.Fatalf("expected running")
	}
	if pct < 11.0 || pct > 13.0 {
		t.Fatalf("unexpected pct: %f", pct)
	}
}

func TestBalanceStatusNotRunning(t *testing.T) {
	out := "No balance found on '/mnt/p'"
	run, pct := balanceStatus(out)
	if run || pct != 0 {
		t.Fatalf("expected not running")
	}
}

func TestReplaceStatusHeuristic(t *testing.T) {
	if s := replaceStatus("State: running"); s != "running" {
		t.Fatalf("got %s", s)
	}
	if s := replaceStatus("Finished"); s != "finished" {
		t.Fatalf("got %s", s)
	}
}

func TestParseBalanceInfoVariants(t *testing.T) {
	out := "Balance on '/mnt/p': running\n  120 out of about 1000 chunks balanced (123 considered),  88% left\n  12% done, 0 errors\n"
	info := parseBalanceInfo(out)
	if !info.Running || info.Percent < 11 || info.Left != "880" || info.Total != "1000" {
		t.Fatalf("unexpected info: %+v", info)
	}
	out2 := "No balance found on '/mnt/p'\n"
	info2 := parseBalanceInfo(out2)
	if info2.Running || info2.Percent != 0 {
		t.Fatalf("expected idle: %+v", info2)
	}
}

func TestParseReplaceInfo(t *testing.T) {
	out := "Replace on '/mnt/p': running\n  3/30  10% done\n"
	ri := parseReplaceInfo(out)
	if !ri.Running || ri.Completed != 3 || ri.Total != 30 || ri.Percent < 9 {
		t.Fatalf("unexpected replace info: %+v", ri)
	}
	out2 := "Replace on '/mnt/p': finished\n  30/30  100% done\n"
	ri2 := parseReplaceInfo(out2)
	if ri2.Running || ri2.Completed != 30 || ri2.Total != 30 || ri2.Percent < 99 {
		t.Fatalf("unexpected replace info finished: %+v", ri2)
	}
	out3 := "Replace on '/mnt/p': not started\n"
	ri3 := parseReplaceInfo(out3)
	if ri3.Running {
		t.Fatalf("expected not running")
	}
}
