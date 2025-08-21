package hash

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestHashAndVerify_Success(t *testing.T) {
	phc, err := HashPassword("s3cret!")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if !VerifyPassword(phc, "s3cret!") {
		t.Fatal("expected verify success")
	}
}

func TestHashAndVerify_Fail(t *testing.T) {
	phc, err := HashPassword("password")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if VerifyPassword(phc, "wrong") {
		t.Fatal("expected verify failure for wrong password")
	}
}

func TestPHCParsing(t *testing.T) {
	// build a minimal valid PHC with known salt/hash lengths
	salt := make([]byte, defaultSaltLen)
	for i := range salt {
		salt[i] = byte(i)
	}
	sum := make([]byte, defaultKeyLen)
	for i := range sum {
		sum[i] = byte(i)
	}
	phc := strings.Join([]string{
		"",
		phcAlg,
		"v=19",
		"m=65536,t=3,p=1",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	}, "$")
	p, s, h, err := parsePHC(phc)
	if err != nil {
		t.Fatalf("parse phc: %v", err)
	}
	if p.memory != defaultMemory || p.time != defaultTime || p.threads != defaultThreads {
		t.Fatalf("params mismatch: %+v", p)
	}
	if len(s) != int(defaultSaltLen) || len(h) != int(defaultKeyLen) {
		t.Fatalf("decoded lengths wrong: salt=%d hash=%d", len(s), len(h))
	}
}
