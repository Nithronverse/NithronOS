package auth

import (
	"fmt"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func GenerateTOTPSecret(issuer, account string) (secret, otpauth string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

func VerifyTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

func OTPAuthURI(issuer, account, secret string) string {
	// fallback if needed
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30", issuer, account, secret, issuer)
}
