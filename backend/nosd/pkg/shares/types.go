package shares

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ShareNameRegex validates share names
var ShareNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-_]{1,31}$`)

// Share represents a network share configuration
type Share struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	SMB         *SMBConfig `json:"smb,omitempty"`
	NFS         *NFSConfig `json:"nfs,omitempty"`
	Owners      []string   `json:"owners,omitempty"`  // users/groups with rwx
	Readers     []string   `json:"readers,omitempty"` // users/groups with rx
	Description string     `json:"description,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

// SMBConfig represents SMB/CIFS share configuration
type SMBConfig struct {
	Enabled     bool           `json:"enabled"`
	Guest       bool           `json:"guest"`
	TimeMachine bool           `json:"time_machine"`
	Recycle     *RecycleConfig `json:"recycle,omitempty"`
}

// RecycleConfig represents recycle bin configuration
type RecycleConfig struct {
	Enabled   bool   `json:"enabled"`
	Directory string `json:"directory,omitempty"` // defaults to .recycle
}

// NFSConfig represents NFS export configuration
type NFSConfig struct {
	Enabled  bool     `json:"enabled"`
	Networks []string `json:"networks,omitempty"` // CIDR blocks, defaults to LAN
	ReadOnly bool     `json:"read_only"`
}

// SharesFile represents the persisted shares configuration
type SharesFile struct {
	Version int      `json:"version"`
	Items   []*Share `json:"items"`
}

// Validate checks if a share configuration is valid
func (s *Share) Validate() error {
	// Validate name
	if !ShareNameRegex.MatchString(s.Name) {
		return fmt.Errorf("invalid share name: must match %s", ShareNameRegex.String())
	}

	// Validate path (will be under /srv/shares/<name>)
	if s.Path == "" {
		s.Path = fmt.Sprintf("/srv/shares/%s", s.Name)
	}

	// At least one protocol must be enabled
	smbEnabled := s.SMB != nil && s.SMB.Enabled
	nfsEnabled := s.NFS != nil && s.NFS.Enabled
	if !smbEnabled && !nfsEnabled {
		return fmt.Errorf("at least one protocol (SMB or NFS) must be enabled")
	}

	// Validate owners/readers format (user:username or group:groupname)
	for _, owner := range s.Owners {
		if !isValidPrincipal(owner) {
			return fmt.Errorf("invalid owner format: %s (use user:name or group:name)", owner)
		}
	}
	for _, reader := range s.Readers {
		if !isValidPrincipal(reader) {
			return fmt.Errorf("invalid reader format: %s (use user:name or group:name)", reader)
		}
	}

	// Set default recycle directory
	if s.SMB != nil && s.SMB.Recycle != nil && s.SMB.Recycle.Enabled {
		if s.SMB.Recycle.Directory == "" {
			s.SMB.Recycle.Directory = ".recycle"
		}
	}

	return nil
}

// isValidPrincipal checks if a principal is in format user:name or group:name
func isValidPrincipal(principal string) bool {
	parts := strings.Split(principal, ":")
	if len(parts) != 2 {
		return false
	}
	if parts[0] != "user" && parts[0] != "group" {
		return false
	}
	// Basic validation for username/groupname
	if matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9-_]{0,31}$`, parts[1]); !matched {
		return false
	}
	return true
}

// CreateRequest represents a share creation request
type CreateRequest struct {
	Name        string     `json:"name"`
	SMB         *SMBConfig `json:"smb,omitempty"`
	NFS         *NFSConfig `json:"nfs,omitempty"`
	Owners      []string   `json:"owners,omitempty"`
	Readers     []string   `json:"readers,omitempty"`
	Description string     `json:"description,omitempty"`
}

// UpdateRequest represents a share update request
type UpdateRequest struct {
	SMB         *SMBConfig `json:"smb,omitempty"`
	NFS         *NFSConfig `json:"nfs,omitempty"`
	Owners      []string   `json:"owners,omitempty"`
	Readers     []string   `json:"readers,omitempty"`
	Description *string    `json:"description,omitempty"`
}

// TestRequest represents a dry-run test request
type TestRequest struct {
	Config json.RawMessage `json:"config"`
}

// TestResponse represents the result of a dry-run test
type TestResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ErrorCode represents specific error codes for share operations
type ErrorCode string

const (
	ErrCodeInvalidName      ErrorCode = "share.name.invalid"
	ErrCodeNameExists       ErrorCode = "share.name.exists"
	ErrCodePathExists       ErrorCode = "share.path.exists"
	ErrCodeSMBConfigInvalid ErrorCode = "smb.config.invalid"
	ErrCodeNFSExportFail    ErrorCode = "nfs.export.fail"
	ErrCodeACLApplyFail     ErrorCode = "acl.apply.fail"
	ErrCodeServiceReload    ErrorCode = "service.reload.fail"
	ErrCodePermission       ErrorCode = "permission.denied"
	ErrCodeNotFound         ErrorCode = "share.not.found"
)

// Error represents a structured error response
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details any       `json:"details,omitempty"`
}
