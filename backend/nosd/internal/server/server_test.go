package server

import "os"

func init() {
	// Skip auth in handler tests
	os.Setenv("NOS_TEST_SKIP_AUTH", "1")
	// Soften rate limits to avoid delays in tests
	os.Setenv("NOS_RATE_OTP_PER_MIN", "1000")
	os.Setenv("NOS_RATE_LOGIN_PER_15M", "1000")
	os.Setenv("NOS_RATE_OTP_WINDOW_SEC", "1")
	os.Setenv("NOS_RATE_LOGIN_WINDOW_SEC", "1")
}
