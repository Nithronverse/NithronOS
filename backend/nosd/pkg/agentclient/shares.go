package agentclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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
func (c *Client) CreateShare(req *CreateShareRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/create", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// ApplyACLs applies POSIX ACLs to a share
func (c *Client) ApplyACLs(req *ApplyACLsRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/acls", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// WriteSambaConfig writes a Samba configuration snippet
func (c *Client) WriteSambaConfig(req *WriteSambaConfigRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/samba/write", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// RemoveSambaConfig removes a Samba configuration snippet
func (c *Client) RemoveSambaConfig(name string) error {
	req := map[string]string{"name": name}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/samba/remove", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// ReloadSamba reloads the Samba service
func (c *Client) ReloadSamba() error {
	resp, err := c.doRequest("POST", "/shares/samba/reload", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// WriteNFSExport writes an NFS export configuration
func (c *Client) WriteNFSExport(req *WriteNFSExportRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/nfs/write", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// RemoveNFSExport removes an NFS export configuration
func (c *Client) RemoveNFSExport(name string) error {
	req := map[string]string{"name": name}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/nfs/remove", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// ReloadNFS reloads the NFS exports
func (c *Client) ReloadNFS() error {
	resp, err := c.doRequest("POST", "/shares/nfs/reload", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// CreateSubvol creates a Btrfs subvolume if the filesystem supports it
func (c *Client) CreateSubvol(req *CreateSubvolRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/subvol", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// EnsureGroup ensures a system group exists
func (c *Client) EnsureGroup(name string) error {
	req := map[string]string{"name": name}
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doRequest("POST", "/shares/group", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}

// ReloadAvahi reloads the Avahi daemon
func (c *Client) ReloadAvahi() error {
	resp, err := c.doRequest("POST", "/shares/avahi/reload", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}

	return nil
}
