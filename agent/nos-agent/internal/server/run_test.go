package server

import "testing"

func TestAllowedBtrfsPrefix(t *testing.T) {
	if allowedBtrfsPrefix([]string{"balance", "status"}) != true {
		t.Fatalf("expected allow balance status")
	}
	if allowedBtrfsPrefix([]string{";", "rm", "-rf", "/"}) {
		t.Fatalf("should not allow shell injection prefix")
	}
	if allowedBtrfsPrefix([]string{"weird", "verb"}) {
		t.Fatalf("should not allow unknown verb")
	}
}

func TestIsAllowedMountPath(t *testing.T) {
	if !isAllowedMountPath("/srv/pool/x") {
		t.Fatalf("expected allowed")
	}
	if !isAllowedMountPath("/mnt/pool/x") {
		t.Fatalf("expected allowed")
	}
	if isAllowedMountPath("/etc/passwd") {
		t.Fatalf("should reject non srv/mnt")
	}
	if isAllowedMountPath("relative") {
		t.Fatalf("should reject relative")
	}
}

func TestAllowedCommandBalanceStatus(t *testing.T) {
	if !allowedCommand("btrfs", []string{"balance", "status", "/srv/pool/x"}) {
		t.Fatalf("expected allowed")
	}
	if allowedCommand("btrfs", []string{"balance", "status", "../../etc"}) {
		t.Fatalf("should reject relative path")
	}
}
