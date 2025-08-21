package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

// encryptWithSecretKey encrypts plaintext using XChaCha20-Poly1305 with the 32-byte key at secretPath.
// Returns base64(nonce||ciphertext) string.
func encryptWithSecretKey(secretPath string, plaintext []byte) (string, error) {
	key, err := os.ReadFile(secretPath)
	if err != nil {
		return "", err
	}
	if len(key) < chacha20poly1305.KeySize {
		return "", errors.New("secret key too short")
	}
	aead, err := chacha20poly1305.NewX(key[:chacha20poly1305.KeySize])
	if err != nil {
		return "", err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := aead.Seal(nil, nonce, plaintext, nil)
	blob := append(nonce, ct...)
	return base64.RawStdEncoding.EncodeToString(blob), nil
}

// decryptWithSecretKey reverses encryptWithSecretKey.
func decryptWithSecretKey(secretPath, b64 string) ([]byte, error) {
	key, err := os.ReadFile(secretPath)
	if err != nil {
		return nil, err
	}
	if len(key) < chacha20poly1305.KeySize {
		return nil, errors.New("secret key too short")
	}
	aead, err := chacha20poly1305.NewX(key[:chacha20poly1305.KeySize])
	if err != nil {
		return nil, err
	}
	blob, err := base64.RawStdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	if len(blob) < chacha20poly1305.NonceSizeX {
		return nil, errors.New("ciphertext too short")
	}
	nonce := blob[:chacha20poly1305.NonceSizeX]
	ct := blob[chacha20poly1305.NonceSizeX:]
	pt, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

// hashRecovery returns hex-encoded SHA-256 of the input.
func hashRecovery(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

// generateRecoveryCodes returns 10 plaintext codes and their SHA256 hex hashes.
func generateRecoveryCodes() (plaintext []string, hashes []string) {
	// 10 codes, 10 characters each from URL-safe base64 (trim padding)
	plaintext = make([]string, 10)
	hashes = make([]string, 10)
	for i := 0; i < 10; i++ {
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		// 8 bytes -> ~11 chars; take 10
		s := base64.RawURLEncoding.EncodeToString(b)
		if len(s) > 10 {
			s = s[:10]
		} else {
			// pad with time nibble if needed (unlikely)
			s = (s + hex.EncodeToString([]byte{byte(time.Now().UnixNano())}))[:10]
		}
		plaintext[i] = s
		hashes[i] = hashRecovery(s)
	}
	return
}
