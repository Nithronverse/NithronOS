package server

import (
    "os"
    "strings"
    "testing"
)

// This is a lightweight assertion that the shipped unit contains the
// sandbox directives we expect. It doesn't require systemd.
func TestNosdServiceUnitContainsSandboxing(t *testing.T) {
    path := "../../../deploy/systemd/nosd.service"
    b, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("read unit: %v", err)
    }
    s := string(b)
    for _, want := range []string{
        "ProtectSystem=strict",
        "NoNewPrivileges=yes",
        "ReadWritePaths=/etc/nos /var/lib/nos /run",
        "StateDirectory=nos",
        "ConfigurationDirectory=nos",
        "User=nosd",
        "Group=nosd",
    } {
        if !strings.Contains(s, want) {
            t.Fatalf("unit missing %q", want)
        }
    }
}


