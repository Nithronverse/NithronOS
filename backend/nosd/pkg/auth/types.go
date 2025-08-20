package auth

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	PasswordHash string    `json:"password_hash"`
	TOTPSecret   string    `json:"totp_secret,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	Roles        []string  `json:"roles"`
}

type Session struct {
	UserID      string   `json:"uid"`
	Roles       []string `json:"roles"`
	TwoFA       bool     `json:"2fa_ok"`
	PendingTOTP string   `json:"pending_totp,omitempty"`
}
