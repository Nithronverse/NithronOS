package shares

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// SambaTemplate generates SMB configuration for a share
const sambaTemplate = `[{{.Name}}]
  path = {{.Path}}
  browseable = yes
  writable = yes
  create mask = 0660
  directory mask = 2770
  ea support = yes
  guest ok = {{if .Guest}}yes{{else}}no{{end}}
  map acl inherit = yes
  inherit acls = yes
  vfs objects = catia streams_xattr{{if .Recycle}} recycle{{end}}{{if .TimeMachine}} fruit{{end}}
{{if .Recycle}}
  # Recycle bin
  recycle:repository = {{.RecycleDir}}
  recycle:keeptree = yes
  recycle:versions = yes
  recycle:touch = yes
  recycle:directory_mode = 0770
  recycle:subdir_mode = 0700
{{end}}
{{if .TimeMachine}}
  # Time Machine support
  fruit:time machine = yes
  fruit:metadata = stream
  fruit:resource = stream
  fruit:posix_rename = yes
  fruit:zero_file_id = yes
  fruit:model = MacSamba
  durable handles = yes
  kernel oplocks = no
  posix locking = no
{{end}}
{{if .Comment}}
  comment = {{.Comment}}
{{end}}
`

// NFSTemplate generates NFS export configuration
const nfsTemplate = `# NithronOS NFS Export: {{.Name}}
{{.Path}} {{range .Networks}}{{.}}({{$.Options}}) {{end}}
`

// GenerateSambaConfig creates a Samba configuration for a share
func GenerateSambaConfig(share *Share) (string, error) {
	if share.SMB == nil || !share.SMB.Enabled {
		return "", fmt.Errorf("smb not enabled for share %s", share.Name)
	}

	tmpl, err := template.New("samba").Parse(sambaTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Samba template: %w", err)
	}

	data := struct {
		Name        string
		Path        string
		Guest       bool
		Recycle     bool
		RecycleDir  string
		TimeMachine bool
		Comment     string
	}{
		Name:        sanitizeSambaValue(share.Name),
		Path:        share.Path,
		Guest:       share.SMB.Guest,
		Recycle:     share.SMB.Recycle != nil && share.SMB.Recycle.Enabled,
		RecycleDir:  ".recycle",
		TimeMachine: share.SMB.TimeMachine,
		Comment:     sanitizeSambaValue(share.Description),
	}

	if share.SMB.Recycle != nil && share.SMB.Recycle.Directory != "" {
		data.RecycleDir = sanitizeSambaValue(share.SMB.Recycle.Directory)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute Samba template: %w", err)
	}

	return buf.String(), nil
}

// GenerateNFSExport creates an NFS export configuration for a share
func GenerateNFSExport(share *Share, lanNetworks []string) (string, error) {
	if share.NFS == nil || !share.NFS.Enabled {
		return "", fmt.Errorf("nfs not enabled for share %s", share.Name)
	}

	// Use configured networks or default to LAN
	networks := share.NFS.Networks
	if len(networks) == 0 {
		if len(lanNetworks) > 0 {
			networks = lanNetworks
		} else {
			// Fallback to restrictive default
			networks = []string{"*"}
		}
	}

	// Build NFS options
	options := []string{"sec=sys"}

	if share.NFS.ReadOnly || (share.SMB != nil && share.SMB.Guest && len(share.Owners) == 0) {
		options = append(options, "ro")
	} else {
		options = append(options, "rw", "sync")
	}

	// Security options
	options = append(options, "root_squash", "all_squash")
	options = append(options, "anonuid=65534", "anongid=65534") // nobody:nogroup

	tmpl, err := template.New("nfs").Parse(nfsTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse NFS template: %w", err)
	}

	data := struct {
		Name     string
		Path     string
		Networks []string
		Options  string
	}{
		Name:     share.Name,
		Path:     share.Path,
		Networks: networks,
		Options:  strings.Join(options, ","),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute NFS template: %w", err)
	}

	return buf.String(), nil
}

// sanitizeSambaValue escapes special characters for Samba config values
func sanitizeSambaValue(value string) string {
	// Samba uses backslash escaping for special characters
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	)
	return replacer.Replace(value)
}

// GetSambaConfigPath returns the path for a share's Samba config
func GetSambaConfigPath(shareName string) string {
	return fmt.Sprintf("/etc/samba/smb.conf.d/nos-%s.conf", shareName)
}

// GetNFSExportPath returns the path for a share's NFS export
func GetNFSExportPath(shareName string) string {
	return fmt.Sprintf("/etc/exports.d/nos-%s.exports", shareName)
}
