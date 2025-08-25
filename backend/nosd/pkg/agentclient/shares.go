package agentclient

import (
	"context"
)

// Share-related request/response types

// CreateShareRequest represents a request to create a share directory
type CreateShareRequest struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Owners     []string `json:"owners"`
	Readers    []string `json:"readers"`
	Mode       uint32   `json:"mode,omitempty"`        // defaults to 02770
	RecycleDir string   `json:"recycle_dir,omitempty"` // for recycle bin
}

// ApplyACLsRequest represents a request to apply POSIX ACLs
type ApplyACLsRequest struct {
	Path    string   `json:"path"`
	Owners  []string `json:"owners"`  // users/groups with rwx
	Readers []string `json:"readers"` // users/groups with rx
}

// WriteSambaConfigRequest represents a request to write Samba config
type WriteSambaConfigRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// WriteNFSExportRequest represents a request to write NFS export
type WriteNFSExportRequest struct {
	Name   string `json:"name"`
	Config string `json:"config"`
}

// CreateSubvolRequest represents a request to create a Btrfs subvolume
type CreateSubvolRequest struct {
	Path string `json:"path"`
}

// Share management methods

// CreateShare creates a share directory with proper permissions
func (c *Client) CreateShare(ctx context.Context, req *CreateShareRequest) error {
	return c.PostJSON(ctx, "/shares/create", req, nil)
}

// ApplyACLs applies POSIX ACLs to a share
func (c *Client) ApplyACLs(ctx context.Context, req *ApplyACLsRequest) error {
	return c.PostJSON(ctx, "/shares/acls", req, nil)
}

// WriteSambaConfig writes Samba configuration for a share
func (c *Client) WriteSambaConfig(ctx context.Context, req *WriteSambaConfigRequest) error {
	return c.PostJSON(ctx, "/shares/samba", req, nil)
}

// WriteNFSExport writes NFS export configuration for a share
func (c *Client) WriteNFSExport(ctx context.Context, req *WriteNFSExportRequest) error {
	return c.PostJSON(ctx, "/shares/nfs", req, nil)
}

// ReloadSamba reloads Samba services
func (c *Client) ReloadSamba(ctx context.Context) error {
	return c.PostJSON(ctx, "/shares/reload-samba", nil, nil)
}

// ReloadNFS reloads NFS exports
func (c *Client) ReloadNFS(ctx context.Context) error {
	return c.PostJSON(ctx, "/shares/reload-nfs", nil, nil)
}

// EnsureGroup ensures a system group exists
func (c *Client) EnsureGroup(ctx context.Context, name string) error {
	req := struct {
		Name string `json:"name"`
	}{
		Name: name,
	}
	return c.PostJSON(ctx, "/shares/ensure-group", req, nil)
}

// CreateSubvol creates a Btrfs subvolume (if the filesystem is Btrfs)
func (c *Client) CreateSubvol(ctx context.Context, req *CreateSubvolRequest) error {
	return c.PostJSON(ctx, "/shares/subvol", req, nil)
}

// TestSambaConfig tests the Samba configuration
func (c *Client) TestSambaConfig(ctx context.Context) error {
	return c.GetJSON(ctx, "/shares/test-samba", nil)
}

// TestNFSExports tests the NFS exports configuration
func (c *Client) TestNFSExports(ctx context.Context) error {
	return c.GetJSON(ctx, "/shares/test-nfs", nil)
}

// WriteAvahiService writes an Avahi service file for Time Machine
func (c *Client) WriteAvahiService(ctx context.Context, name string, content string) error {
	req := struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}{
		Name:    name,
		Content: content,
	}
	return c.PostJSON(ctx, "/shares/avahi", req, nil)
}

// RemoveAvahiService removes an Avahi service file
func (c *Client) RemoveAvahiService(ctx context.Context, name string) error {
	// For now, use POST with a delete action
	req := struct {
		Name   string `json:"name"`
		Action string `json:"action"`
	}{
		Name:   name,
		Action: "delete",
	}
	return c.PostJSON(ctx, "/shares/avahi", req, nil)
}

// ReloadAvahi reloads the Avahi daemon
func (c *Client) ReloadAvahi(ctx context.Context) error {
	return c.PostJSON(ctx, "/shares/reload-avahi", nil, nil)
}
