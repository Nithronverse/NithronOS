package server

import (
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// prepare temp key
	f, err := os.CreateTemp(t.TempDir(), "key")
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	if err := os.WriteFile(f.Name(), key, 0o600); err != nil {
		t.Fatal(err)
	}
	// Close the file so TempDir cleanup on Windows can remove it
	_ = f.Close()

	msg := []byte("hello-world")
	ct, err := encryptWithSecretKey(f.Name(), msg)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pt, err := decryptWithSecretKey(f.Name(), ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(pt) != string(msg) {
		t.Fatalf("roundtrip mismatch: %q != %q", pt, msg)
	}
}

func TestGenerateRecoveryCodes(t *testing.T) {
	plain, hashes := generateRecoveryCodes()
	if len(plain) != 10 || len(hashes) != 10 {
		t.Fatal("expected 10 codes")
	}
	for i := 0; i < 10; i++ {
		if len(plain[i]) != 10 {
			t.Fatal("code length")
		}
		if hashes[i] != hashRecovery(plain[i]) {
			t.Fatal("hash mismatch")
		}
	}
}
