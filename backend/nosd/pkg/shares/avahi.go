package shares

import (
	"fmt"
	"os"
	"path/filepath"
)

const avahiServicePath = "/etc/avahi/services/nithronos-tm.service"

// AvahiTimeMachineService generates an Avahi service file for Time Machine discovery
const avahiTimeMachineTemplate = `<?xml version="1.0" standalone='no'?>
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">
<service-group>
  <name>NithronOS Time Machine</name>
  
  <!-- SMB service for Time Machine -->
  <service>
    <type>_smb._tcp</type>
    <port>445</port>
  </service>
  
  <!-- Apple Disk service for Time Machine discovery -->
  <service>
    <type>_adisk._tcp</type>
    <port>9</port>
    <txt-record>dk0=adVN=Time Machine,adVF=0x82</txt-record>
  </service>
  
  <!-- Device info -->
  <service>
    <type>_device-info._tcp</type>
    <port>0</port>
    <txt-record>model=TimeCapsule8,119</txt-record>
  </service>
</service-group>
`

// UpdateAvahiTimeMachine updates the Avahi service file for Time Machine if needed
func UpdateAvahiTimeMachine(shares []*Share) error {
	// Check if any share has Time Machine enabled
	hasTimeMachine := false
	for _, share := range shares {
		if share.SMB != nil && share.SMB.Enabled && share.SMB.TimeMachine {
			hasTimeMachine = true
			break
		}
	}

	// Get current state
	_, err := os.Stat(avahiServicePath)
	fileExists := err == nil

	// If we need Time Machine and file doesn't exist, create it
	if hasTimeMachine && !fileExists {
		// Ensure directory exists
		dir := filepath.Dir(avahiServicePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create Avahi services directory: %w", err)
		}

		// Write service file
		if err := os.WriteFile(avahiServicePath, []byte(avahiTimeMachineTemplate), 0644); err != nil {
			return fmt.Errorf("failed to write Avahi service file: %w", err)
		}

		return nil // Caller should reload Avahi
	}

	// If we don't need Time Machine and file exists, remove it
	if !hasTimeMachine && fileExists {
		if err := os.Remove(avahiServicePath); err != nil {
			return fmt.Errorf("failed to remove Avahi service file: %w", err)
		}

		return nil // Caller should reload Avahi
	}

	// No change needed
	return nil
}

// GenerateAvahiShareService generates an Avahi service for a specific Time Machine share
func GenerateAvahiShareService(share *Share) string {
	if share.SMB == nil || !share.SMB.Enabled || !share.SMB.TimeMachine {
		return ""
	}

	return fmt.Sprintf(`<?xml version="1.0" standalone='no'?>
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">
<service-group>
  <name>%s - Time Machine</name>
  
  <service>
    <type>_smb._tcp</type>
    <port>445</port>
    <txt-record>path=%s</txt-record>
  </service>
  
  <service>
    <type>_adisk._tcp</type>
    <port>9</port>
    <txt-record>dk0=adVN=%s,adVF=0x82</txt-record>
    <txt-record>sys=adVF=0x100</txt-record>
  </service>
</service-group>
`, share.Name, share.Path, share.Name)
}
