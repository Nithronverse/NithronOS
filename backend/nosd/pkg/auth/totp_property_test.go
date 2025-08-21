package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// Property: acceptance window limited to ±1 step (30s)
func TestTOTPWindowPlusMinusOne(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Test",
		AccountName: "user",
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		t.Fatal(err)
	}
	secret := key.Secret()

	now := time.Now()
	// Current and ±1 window should validate
	curr, _ := totp.GenerateCode(secret, now)
	prev, _ := totp.GenerateCode(secret, now.Add(-30*time.Second))
	next, _ := totp.GenerateCode(secret, now.Add(30*time.Second))
	if !totp.Validate(curr, secret) {
		t.Fatalf("current code must validate")
	}
	if !totp.Validate(prev, secret) {
		t.Fatalf("previous window code must validate")
	}
	if !totp.Validate(next, secret) {
		t.Fatalf("next window code must validate")
	}

	// ±2 windows should not validate under default
	prev2, _ := totp.GenerateCode(secret, now.Add(-60*time.Second))
	next2, _ := totp.GenerateCode(secret, now.Add(60*time.Second))
	if totp.Validate(prev2, secret) {
		t.Fatalf("-2 window code must not validate")
	}
	if totp.Validate(next2, secret) {
		t.Fatalf("+2 window code must not validate")
	}
}
