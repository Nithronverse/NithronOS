package net

import (
	"testing"
	"time"
	
	"github.com/pquerna/otp/totp"
)

func TestTOTPManager_EnrollAndVerify(t *testing.T) {
	tm := NewTOTPManager()
	
	// Test enrollment
	userID := "test-user-1"
	username := "testuser"
	
	enrollment, err := tm.EnrollUser(userID, username)
	if err != nil {
		t.Fatalf("Failed to enroll user: %v", err)
	}
	
	// Check enrollment data
	if enrollment.Secret == "" {
		t.Error("Expected secret to be non-empty")
	}
	if enrollment.QRCode == "" {
		t.Error("Expected QR code to be non-empty")
	}
	if len(enrollment.BackupCodes) != backupCodeCount {
		t.Errorf("Expected %d backup codes, got %d", backupCodeCount, len(enrollment.BackupCodes))
	}
	if enrollment.URI == "" {
		t.Error("Expected URI to be non-empty")
	}
	
	// User should not be enrolled until first verification
	if tm.IsUserEnrolled(userID) {
		t.Error("User should not be enrolled before verification")
	}
	
	// Test verification with valid code
	code, err := totp.GenerateCode(enrollment.Secret, time.Now())
	if err != nil {
		t.Fatalf("Failed to generate TOTP code: %v", err)
	}
	
	valid, err := tm.VerifyCode(userID, code)
	if err != nil {
		t.Fatalf("Failed to verify code: %v", err)
	}
	if !valid {
		t.Error("Expected valid code to be accepted")
	}
	
	// User should now be enrolled
	if !tm.IsUserEnrolled(userID) {
		t.Error("User should be enrolled after verification")
	}
	
	// Test verification with invalid code
	valid, err = tm.VerifyCode(userID, "000000")
	if err != nil {
		t.Fatalf("Failed to verify invalid code: %v", err)
	}
	if valid {
		t.Error("Expected invalid code to be rejected")
	}
	
	// Test backup code verification
	backupCode := enrollment.BackupCodes[0]
	valid, err = tm.VerifyCode(userID, backupCode)
	if err != nil {
		t.Fatalf("Failed to verify backup code: %v", err)
	}
	if !valid {
		t.Error("Expected backup code to be accepted")
	}
	
	// Backup code should not work twice
	valid, err = tm.VerifyCode(userID, backupCode)
	if err != nil {
		t.Fatalf("Failed to verify used backup code: %v", err)
	}
	if valid {
		t.Error("Expected used backup code to be rejected")
	}
}

func TestTOTPManager_SessionManagement(t *testing.T) {
	tm := NewTOTPManager()
	
	sessionID := "test-session-1"
	
	// Session should not be verified initially
	if tm.IsSessionVerified(sessionID) {
		t.Error("Session should not be verified initially")
	}
	
	// Mark session as verified
	tm.MarkSessionVerified(sessionID, 30*time.Minute)
	
	// Session should be verified
	if !tm.IsSessionVerified(sessionID) {
		t.Error("Session should be verified after marking")
	}
	
	// Clear session
	tm.ClearSession(sessionID)
	
	// Session should not be verified after clearing
	if tm.IsSessionVerified(sessionID) {
		t.Error("Session should not be verified after clearing")
	}
	
	// Test expired session
	tm.MarkSessionVerified(sessionID, -1*time.Minute) // Already expired
	if tm.IsSessionVerified(sessionID) {
		t.Error("Expired session should not be verified")
	}
}

func TestTOTPManager_DisableUser(t *testing.T) {
	tm := NewTOTPManager()
	
	userID := "test-user-2"
	username := "testuser2"
	
	// Enroll and verify user
	enrollment, err := tm.EnrollUser(userID, username)
	if err != nil {
		t.Fatalf("Failed to enroll user: %v", err)
	}
	
	code, _ := totp.GenerateCode(enrollment.Secret, time.Now())
	tm.VerifyCode(userID, code)
	
	// User should be enrolled
	if !tm.IsUserEnrolled(userID) {
		t.Error("User should be enrolled")
	}
	
	// Disable user
	err = tm.DisableUser(userID)
	if err != nil {
		t.Fatalf("Failed to disable user: %v", err)
	}
	
	// User should not be enrolled after disabling
	if tm.IsUserEnrolled(userID) {
		t.Error("User should not be enrolled after disabling")
	}
}

func TestRequiresTwoFactor(t *testing.T) {
	tm := NewTOTPManager()
	
	userID := "test-user-3"
	username := "testuser3"
	
	// Enroll user
	enrollment, _ := tm.EnrollUser(userID, username)
	code, _ := totp.GenerateCode(enrollment.Secret, time.Now())
	tm.VerifyCode(userID, code)
	
	tests := []struct {
		name     string
		remoteIP string
		want     bool
	}{
		{"LAN IP (192.168.x.x)", "192.168.1.100", false},
		{"LAN IP (10.x.x.x)", "10.0.0.5", false},
		{"LAN IP (172.16.x.x)", "172.16.0.1", false},
		{"Loopback", "127.0.0.1", false},
		{"Public IP", "8.8.8.8", true},
		{"Public IP 2", "203.0.113.1", true},
		{"Invalid IP", "invalid", true}, // Should require 2FA if IP is invalid
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RequiresTwoFactor(tt.remoteIP, userID, tm)
			if got != tt.want {
				t.Errorf("RequiresTwoFactor(%s) = %v, want %v", tt.remoteIP, got, tt.want)
			}
		})
	}
	
	// Test with non-enrolled user
	nonEnrolledUser := "test-user-4"
	for _, tt := range tests {
		t.Run(tt.name+" (non-enrolled)", func(t *testing.T) {
			got := RequiresTwoFactor(tt.remoteIP, nonEnrolledUser, tm)
			// Non-enrolled users should not require 2FA regardless of IP
			if got {
				t.Errorf("RequiresTwoFactor(%s) for non-enrolled user = %v, want false", tt.remoteIP, got)
			}
		})
	}
}
