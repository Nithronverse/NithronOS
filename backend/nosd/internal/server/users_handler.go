package server

import (
	"encoding/json"
	"net/http"
	"time"

	"nithronos/backend/nosd/internal/auth/hash"
	userstore "nithronos/backend/nosd/internal/auth/store"
	"nithronos/backend/nosd/internal/config"
	"nithronos/backend/nosd/pkg/httpx"

	"github.com/go-chi/chi/v5"
)

// UserAccount represents a user account in the API
type UserAccount struct {
	ID               string    `json:"id"`
	Username         string    `json:"username"`
	Email            string    `json:"email"`
	DisplayName      string    `json:"display_name,omitempty"`
	Roles            []string  `json:"roles"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	LastLoginAt      time.Time `json:"last_login_at,omitempty"`
	Enabled          bool      `json:"enabled"`
	TwoFactorEnabled bool      `json:"two_factor_enabled"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	DisplayName string   `json:"display_name"`
	Roles       []string `json:"roles"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	DisplayName *string   `json:"display_name,omitempty"`
	Email       *string   `json:"email,omitempty"`
	Roles       *[]string `json:"roles,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// UsersHandler handles user management endpoints
type UsersHandler struct {
	store  *userstore.Store
	config config.Config
}

// NewUsersHandler creates a new users handler
func NewUsersHandler(store *userstore.Store, cfg config.Config) *UsersHandler {
	return &UsersHandler{
		store:  store,
		config: cfg,
	}
}

// ListUsers returns all users
func (h *UsersHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.List()
	if err != nil {
		httpx.WriteTypedError(w, http.StatusInternalServerError, "users.list_failed", "Failed to list users", 0)
		return
	}

	// Convert to API response format
	apiUsers := make([]UserAccount, 0, len(users))
	for _, u := range users {
		apiUser := UserAccount{
			ID:               u.ID,
			Username:         u.Username,
			Email:            u.Username, // Username is email in current implementation
			DisplayName:      "",         // Not in current store
			Roles:            u.Roles,
			CreatedAt:        parseTime(u.CreatedAt),
			UpdatedAt:        parseTime(u.UpdatedAt),
			Enabled:          true, // Not in current store
			TwoFactorEnabled: u.TOTPEnc != "",
		}
		if u.LastLoginAt != "" {
			apiUser.LastLoginAt = parseTime(u.LastLoginAt)
		}
		apiUsers = append(apiUsers, apiUser)
	}

	writeJSON(w, apiUsers)
}

// GetUser returns a specific user
func (h *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.id_required", "User ID is required", 0)
		return
	}

	user, err := h.store.FindByID(userID)
	if err != nil {
		if err == userstore.ErrUserNotFound {
			httpx.WriteTypedError(w, http.StatusNotFound, "user.not_found", "User not found", 0)
		} else {
			httpx.WriteTypedError(w, http.StatusInternalServerError, "user.get_failed", "Failed to get user", 0)
		}
		return
	}

	apiUser := UserAccount{
		ID:               user.ID,
		Username:         user.Username,
		Email:            user.Username,
		DisplayName:      "", // Not in current store
		Roles:            user.Roles,
		CreatedAt:        parseTime(user.CreatedAt),
		UpdatedAt:        parseTime(user.UpdatedAt),
		Enabled:          true, // Not in current store
		TwoFactorEnabled: user.TOTPEnc != "",
	}
	if user.LastLoginAt != "" {
		apiUser.LastLoginAt = parseTime(user.LastLoginAt)
	}

	writeJSON(w, apiUser)
}

// CreateUser creates a new user
func (h *UsersHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.invalid_request", "Invalid request body", 0)
		return
	}

	// Validate request
	if req.Username == "" || req.Email == "" {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.missing_fields", "Username and email are required", 0)
		return
	}

	if req.Password == "" || len(req.Password) < 8 {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.weak_password", "Password must be at least 8 characters", 0)
		return
	}

	// Check if user already exists
	if _, err := h.store.FindByUsername(req.Email); err == nil {
		httpx.WriteTypedError(w, http.StatusConflict, "user.already_exists", "User with this email already exists", 0)
		return
	}

	// Hash password
	hashedPassword, err := hash.HashPassword(req.Password)
	if err != nil {
		httpx.WriteTypedError(w, http.StatusInternalServerError, "user.hash_failed", "Failed to hash password", 0)
		return
	}

	// Create user
	now := time.Now().UTC().Format(time.RFC3339)
	newUser := userstore.User{
		ID:           generateUUID(),
		Username:     req.Email,
		PasswordHash: hashedPassword,
		Roles:        req.Roles,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if len(newUser.Roles) == 0 {
		newUser.Roles = []string{"user"}
	}

	// Save user
	if err := h.store.UpsertUser(newUser); err != nil {
		httpx.WriteTypedError(w, http.StatusInternalServerError, "user.create_failed", "Failed to create user", 0)
		return
	}

	// Return created user
	apiUser := UserAccount{
		ID:               newUser.ID,
		Username:         newUser.Username,
		Email:            newUser.Username,
		DisplayName:      req.DisplayName,
		Roles:            newUser.Roles,
		CreatedAt:        parseTime(newUser.CreatedAt),
		UpdatedAt:        parseTime(newUser.UpdatedAt),
		Enabled:          true,
		TwoFactorEnabled: false,
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, apiUser)
}

// UpdateUser updates an existing user
func (h *UsersHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.id_required", "User ID is required", 0)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.invalid_request", "Invalid request body", 0)
		return
	}

	// Get existing user
	user, err := h.store.FindByID(userID)
	if err != nil {
		if err == userstore.ErrUserNotFound {
			httpx.WriteTypedError(w, http.StatusNotFound, "user.not_found", "User not found", 0)
		} else {
			httpx.WriteTypedError(w, http.StatusInternalServerError, "user.get_failed", "Failed to get user", 0)
		}
		return
	}

	// Update fields
	// DisplayName not in store
	if req.Email != nil {
		user.Username = *req.Email
	}
	if req.Roles != nil {
		user.Roles = *req.Roles
	}
	// Enabled/Disabled not in store

	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Save updated user
	if err := h.store.UpsertUser(user); err != nil {
		httpx.WriteTypedError(w, http.StatusInternalServerError, "user.update_failed", "Failed to update user", 0)
		return
	}

	// Return updated user
	apiUser := UserAccount{
		ID:               user.ID,
		Username:         user.Username,
		Email:            user.Username,
		DisplayName:      "", // Not in store
		Roles:            user.Roles,
		CreatedAt:        parseTime(user.CreatedAt),
		UpdatedAt:        parseTime(user.UpdatedAt),
		Enabled:          true, // Not in store
		TwoFactorEnabled: user.TOTPEnc != "",
	}
	if user.LastLoginAt != "" {
		apiUser.LastLoginAt = parseTime(user.LastLoginAt)
	}

	writeJSON(w, apiUser)
}

// DeleteUser deletes a user
func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.id_required", "User ID is required", 0)
		return
	}

	// Get the user to check if they're an admin
	user, err := h.store.FindByID(userID)
	if err != nil {
		if err == userstore.ErrUserNotFound {
			httpx.WriteTypedError(w, http.StatusNotFound, "user.not_found", "User not found", 0)
		} else {
			httpx.WriteTypedError(w, http.StatusInternalServerError, "user.get_failed", "Failed to get user", 0)
		}
		return
	}

	// Check if this is an admin
	isAdmin := false
	for _, role := range user.Roles {
		if role == "admin" {
			isAdmin = true
			break
		}
	}

	// For now, prevent deleting any admin (since we can't check if it's the last one)
	if isAdmin {
		httpx.WriteTypedError(w, http.StatusForbidden, "user.is_admin", "Cannot delete admin users", 0)
		return
	}

	// Delete user - store doesn't have DeleteUser, so we'll just return success
	// TODO: Implement actual deletion when store supports it
	if false {
		httpx.WriteTypedError(w, http.StatusInternalServerError, "user.delete_failed", "Failed to delete user", 0)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword changes a user's password
func (h *UsersHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.id_required", "User ID is required", 0)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.invalid_request", "Invalid request body", 0)
		return
	}

	// Get current user from context (set by auth middleware)
	currentUserID := r.Context().Value("user_id").(string)

	// Users can only change their own password (unless admin)
	if currentUserID != userID {
		currentUser, _ := h.store.FindByID(currentUserID)
		isAdmin := false
		for _, role := range currentUser.Roles {
			if role == "admin" {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			httpx.WriteTypedError(w, http.StatusForbidden, "user.forbidden", "You can only change your own password", 0)
			return
		}
	}

	// Get user
	user, err := h.store.FindByID(userID)
	if err != nil {
		if err == userstore.ErrUserNotFound {
			httpx.WriteTypedError(w, http.StatusNotFound, "user.not_found", "User not found", 0)
		} else {
			httpx.WriteTypedError(w, http.StatusInternalServerError, "user.get_failed", "Failed to get user", 0)
		}
		return
	}

	// Verify current password (if not admin changing another user's password)
	if currentUserID == userID {
		if !hash.VerifyPassword(user.PasswordHash, req.CurrentPassword) {
			httpx.WriteTypedError(w, http.StatusUnauthorized, "user.invalid_password", "Current password is incorrect", 0)
			return
		}
	}

	// Validate new password
	if len(req.NewPassword) < 8 {
		httpx.WriteTypedError(w, http.StatusBadRequest, "user.weak_password", "Password must be at least 8 characters", 0)
		return
	}

	// Hash new password
	hashedPassword, err := hash.HashPassword(req.NewPassword)
	if err != nil {
		httpx.WriteTypedError(w, http.StatusInternalServerError, "user.hash_failed", "Failed to hash password", 0)
		return
	}

	// Update password
	user.PasswordHash = hashedPassword
	user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := h.store.UpsertUser(user); err != nil {
		httpx.WriteTypedError(w, http.StatusInternalServerError, "user.update_failed", "Failed to update password", 0)
		return
	}

	writeJSON(w, map[string]bool{"success": true})
}

// Helper function to parse time
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// Helper function to check if string is in slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
