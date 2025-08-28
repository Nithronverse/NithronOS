package net

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

const (
	totpPeriod      = 30 // seconds
	totpDigits      = 6  // code length
	totpSkew        = 1  // allow 1 period skew
	backupCodeCount = 10 // number of backup codes
	backupCodeLen   = 8  // length of each backup code
	issuerName      = "NithronOS"
)

// TOTPManager manages TOTP 2FA for users
type TOTPManager struct {
	mu       sync.RWMutex
	configs  map[string]*TOTPConfig // userID -> config
	sessions map[string]time.Time   // sessionID -> 2FA verified time
}

// NewTOTPManager creates a new TOTP manager
func NewTOTPManager() *TOTPManager {
	return &TOTPManager{
		configs:  make(map[string]*TOTPConfig),
		sessions: make(map[string]time.Time),
	}
}

// EnrollUser enrolls a user for TOTP 2FA
func (tm *TOTPManager) EnrollUser(userID, username string) (*TOTPEnrollment, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if already enrolled
	if config, exists := tm.configs[userID]; exists && config.Enabled {
		return nil, fmt.Errorf("user already enrolled in 2FA")
	}

	// Generate secret
	secret, err := generateTOTPSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	// Generate backup codes
	backupCodes, err := generateBackupCodes(backupCodeCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Generate TOTP URI
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuerName,
		AccountName: username,
		Secret:      []byte(secret),
		Period:      totpPeriod,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	// Generate QR code
	qrCode, err := generateQRCode(key.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Store configuration (not yet enabled)
	config := &TOTPConfig{
		UserID:      userID,
		Secret:      secret,
		BackupCodes: encryptBackupCodes(backupCodes),
		Enabled:     false, // Will be enabled after first successful verification
		EnrolledAt:  time.Now(),
	}
	tm.configs[userID] = config

	// Return enrollment data
	enrollment := &TOTPEnrollment{
		Secret:      secret,
		QRCode:      qrCode,
		BackupCodes: backupCodes,
		URI:         key.URL(),
	}

	return enrollment, nil
}

// VerifyCode verifies a TOTP code for a user
func (tm *TOTPManager) VerifyCode(userID, code string) (bool, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	config, exists := tm.configs[userID]
	if !exists {
		return false, fmt.Errorf("user not enrolled in 2FA")
	}

	// Check if it's a backup code
	if tm.verifyBackupCode(config, code) {
		// Enable 2FA if this is first verification
		if !config.Enabled {
			config.Enabled = true
		}
		now := time.Now()
		config.LastUsed = &now
		return true, nil
	}

	// Verify TOTP code
	valid := totp.Validate(code, config.Secret)
	if !valid {
		// Try with time skew
		for i := -totpSkew; i <= totpSkew; i++ {
			if i == 0 {
				continue
			}
			t := time.Now().Add(time.Duration(i*totpPeriod) * time.Second)
			if isValid, _ := totp.ValidateCustom(code, config.Secret, t, totp.ValidateOpts{
				Period:    totpPeriod,
				Skew:      0,
				Digits:    otp.DigitsSix,
				Algorithm: otp.AlgorithmSHA1,
			}); isValid {
				valid = true
				break
			}
		}
	}

	if valid {
		// Enable 2FA if this is first verification
		if !config.Enabled {
			config.Enabled = true
		}
		now := time.Now()
		config.LastUsed = &now
		return true, nil
	}

	return false, nil
}

// IsUserEnrolled checks if a user has 2FA enabled
func (tm *TOTPManager) IsUserEnrolled(userID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	config, exists := tm.configs[userID]
	return exists && config.Enabled
}

// MarkSessionVerified marks a session as 2FA verified
func (tm *TOTPManager) MarkSessionVerified(sessionID string, duration time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.sessions[sessionID] = time.Now().Add(duration)
}

// IsSessionVerified checks if a session has valid 2FA
func (tm *TOTPManager) IsSessionVerified(sessionID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	expiry, exists := tm.sessions[sessionID]
	if !exists {
		return false
	}

	return time.Now().Before(expiry)
}

// ClearSession removes 2FA verification for a session
func (tm *TOTPManager) ClearSession(sessionID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.sessions, sessionID)
}

// DisableUser disables 2FA for a user
func (tm *TOTPManager) DisableUser(userID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	config, exists := tm.configs[userID]
	if !exists {
		return fmt.Errorf("user not enrolled in 2FA")
	}

	config.Enabled = false

	// Clear all sessions for this user
	// In production, you'd want to track user->session mapping

	return nil
}

// RegenerateBackupCodes regenerates backup codes for a user
func (tm *TOTPManager) RegenerateBackupCodes(userID string) ([]string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	config, exists := tm.configs[userID]
	if !exists {
		return nil, fmt.Errorf("user not enrolled in 2FA")
	}

	// Generate new backup codes
	backupCodes, err := generateBackupCodes(backupCodeCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Update configuration
	config.BackupCodes = encryptBackupCodes(backupCodes)

	return backupCodes, nil
}

// GetUserConfig returns the TOTP configuration for a user (without secrets)
func (tm *TOTPManager) GetUserConfig(userID string) (*TOTPConfig, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	config, exists := tm.configs[userID]
	if !exists {
		return nil, fmt.Errorf("user not enrolled in 2FA")
	}

	// Return copy without secrets
	return &TOTPConfig{
		UserID:     config.UserID,
		Enabled:    config.Enabled,
		EnrolledAt: config.EnrolledAt,
		LastUsed:   config.LastUsed,
	}, nil
}

// Private methods

func (tm *TOTPManager) verifyBackupCode(config *TOTPConfig, code string) bool {
	decryptedCodes := decryptBackupCodes(config.BackupCodes)

	for i, backupCode := range decryptedCodes {
		if backupCode == code {
			// Remove used backup code
			config.BackupCodes = append(config.BackupCodes[:i], config.BackupCodes[i+1:]...)
			return true
		}
	}

	return false
}

// Helper functions

func generateTOTPSecret() (string, error) {
	// Generate 20 bytes of random data (160 bits)
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}

	// Encode as base32
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

func generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)

	for i := 0; i < count; i++ {
		code := make([]byte, backupCodeLen)
		if _, err := rand.Read(code); err != nil {
			return nil, err
		}

		// Convert to alphanumeric string
		codes[i] = strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(code))[:backupCodeLen]
	}

	return codes, nil
}

func generateQRCode(uri string) (string, error) {
	qr, err := qrcode.New(uri, qrcode.Medium)
	if err != nil {
		return "", err
	}

	// Generate PNG image
	pngData, err := qr.PNG(256)
	if err != nil {
		return "", err
	}

	// Encode as base64 data URI
	encoded := base64.StdEncoding.EncodeToString(pngData)
	return fmt.Sprintf("data:image/png;base64,%s", encoded), nil
}

// In production, these would use proper encryption
func encryptBackupCodes(codes []string) []string {
	// TODO: Implement proper encryption with KMS or similar
	return codes
}

func decryptBackupCodes(codes []string) []string {
	// TODO: Implement proper decryption with KMS or similar
	return codes
}

// RequiresTwoFactor checks if 2FA is required for a user based on remote IP
func RequiresTwoFactor(remoteIP string, userID string, tm *TOTPManager) bool {
	// Check if user has 2FA enabled first
	if !tm.IsUserEnrolled(userID) {
		// Non-enrolled users don't require 2FA
		return false
	}

	// Parse IP
	ip := net.ParseIP(remoteIP)
	if ip == nil {
		// If we can't parse the IP, require 2FA to be safe (for enrolled users)
		return true
	}

	// Check if IP is LAN
	if IsLANIP(ip) {
		return false
	}

	// Non-LAN IP for enrolled user requires 2FA
	return true
}
