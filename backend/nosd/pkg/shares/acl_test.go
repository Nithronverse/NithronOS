package shares

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildACLCommands(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		owners   []string
		readers  []string
		want     []string
		wantErr  bool
	}{
		{
			name:    "basic owners and readers",
			path:    "/srv/shares/test",
			owners:  []string{"user:alice", "group:admins"},
			readers: []string{"user:bob", "group:users"},
			want: []string{
				"setfacl -m u:alice:rwx /srv/shares/test",
				"setfacl -m d:u:alice:rwx /srv/shares/test",
				"setfacl -m g:admins:rwx /srv/shares/test",
				"setfacl -m d:g:admins:rwx /srv/shares/test",
				"setfacl -m u:bob:rx /srv/shares/test",
				"setfacl -m d:u:bob:rx /srv/shares/test",
				"setfacl -m g:users:rx /srv/shares/test",
				"setfacl -m d:g:users:rx /srv/shares/test",
			},
		},
		{
			name:    "only owners",
			path:    "/srv/shares/docs",
			owners:  []string{"user:admin"},
			readers: []string{},
			want: []string{
				"setfacl -m u:admin:rwx /srv/shares/docs",
				"setfacl -m d:u:admin:rwx /srv/shares/docs",
			},
		},
		{
			name:    "only readers",
			path:    "/srv/shares/public",
			owners:  []string{},
			readers: []string{"group:everyone"},
			want: []string{
				"setfacl -m g:everyone:rx /srv/shares/public",
				"setfacl -m d:g:everyone:rx /srv/shares/public",
			},
		},
		{
			name:    "invalid principal format",
			path:    "/srv/shares/bad",
			owners:  []string{"alice"}, // Missing type prefix
			readers: []string{},
			wantErr: true,
		},
		{
			name:    "invalid principal type",
			path:    "/srv/shares/bad2",
			owners:  []string{"role:admin"}, // Invalid type
			readers: []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildACLCommands(tt.path, tt.owners, tt.readers)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildACLCommands() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if tt.wantErr {
				return
			}
			
			if len(got) != len(tt.want) {
				t.Errorf("BuildACLCommands() returned %d commands, want %d", len(got), len(tt.want))
				t.Errorf("Got: %v", got)
				t.Errorf("Want: %v", tt.want)
				return
			}
			
			for i, cmd := range got {
				if cmd != tt.want[i] {
					t.Errorf("Command[%d] = %q, want %q", i, cmd, tt.want[i])
				}
			}
		})
	}
}

func TestValidatePrincipal(t *testing.T) {
	tests := []struct {
		principal string
		wantErr   bool
	}{
		{"user:alice", false},
		{"group:admins", false},
		{"user:alice-smith", false},
		{"group:nos-share-docs", false},
		{"alice", true},           // Missing type
		{"user:", true},           // Missing name
		{":alice", true},          // Missing type
		{"role:admin", true},      // Invalid type
		{"user:alice:extra", true}, // Too many parts
		{"user:Alice", true},      // Uppercase not allowed
		{"user:alice smith", true}, // Space not allowed
		{"user:alice@domain", true}, // @ not allowed
	}

	for _, tt := range tests {
		t.Run(tt.principal, func(t *testing.T) {
			err := ValidatePrincipal(tt.principal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrincipal(%q) error = %v, wantErr %v", tt.principal, err, tt.wantErr)
			}
		})
	}
}

func TestEscapeForACL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"alice", "alice"},
		{"alice-smith", "alice-smith"},
		{"alice_smith", "alice_smith"},
		{"alice123", "alice123"},
		{"alice'smith", "alice\\'smith"},
		{"alice\"smith", "alice\\\"smith"},
		{"alice\\smith", "alice\\\\smith"},
		{"alice`cmd`", "alice\\`cmd\\`"},
		{"alice$var", "alice\\$var"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := EscapeForACL(tt.input)
			if got != tt.want {
				t.Errorf("EscapeForACL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// BuildACLCommands generates setfacl commands for the given principals
func BuildACLCommands(path string, owners, readers []string) ([]string, error) {
	var commands []string
	
	// Process owners (rwx)
	for _, owner := range owners {
		if err := ValidatePrincipal(owner); err != nil {
			return nil, err
		}
		
		parts := strings.Split(owner, ":")
		aclType := "u"
		if parts[0] == "group" {
			aclType = "g"
		}
		
		name := EscapeForACL(parts[1])
		
		// Access ACL
		commands = append(commands, 
			"setfacl -m "+aclType+":"+name+":rwx "+path)
		// Default ACL for new files
		commands = append(commands,
			"setfacl -m d:"+aclType+":"+name+":rwx "+path)
	}
	
	// Process readers (rx)
	for _, reader := range readers {
		if err := ValidatePrincipal(reader); err != nil {
			return nil, err
		}
		
		parts := strings.Split(reader, ":")
		aclType := "u"
		if parts[0] == "group" {
			aclType = "g"
		}
		
		name := EscapeForACL(parts[1])
		
		// Access ACL
		commands = append(commands,
			"setfacl -m "+aclType+":"+name+":rx "+path)
		// Default ACL for new files
		commands = append(commands,
			"setfacl -m d:"+aclType+":"+name+":rx "+path)
	}
	
	return commands, nil
}

// ValidatePrincipal checks if a principal string is valid
func ValidatePrincipal(principal string) error {
	parts := strings.Split(principal, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid principal format: %s", principal)
	}
	
	if parts[0] != "user" && parts[0] != "group" {
		return fmt.Errorf("invalid principal type: %s", parts[0])
	}
	
	if parts[1] == "" {
		return fmt.Errorf("empty principal name")
	}
	
	// Check name format (lowercase, alphanumeric, dash, underscore)
	for _, r := range parts[1] {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("invalid character in principal name: %c", r)
		}
	}
	
	return nil
}

// EscapeForACL escapes special characters for shell commands
func EscapeForACL(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`'`, `\'`,
		`"`, `\"`,
		"`", "\\`",
		`$`, `\$`,
	)
	return replacer.Replace(s)
}
