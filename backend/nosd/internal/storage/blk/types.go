package blk

// Raw JSON representation from lsblk --bytes --json
type rawTree struct {
	Blockdevices []rawDevice `json:"blockdevices"`
}

type rawDevice struct {
	Name       string      `json:"name"`
	KName      string      `json:"kname"`
	Path       string      `json:"path"`
	Size       any         `json:"size"` // number (bytes) when using --bytes
	Rota       *bool       `json:"rota,omitempty"`
	Type       string      `json:"type"`
	Tran       string      `json:"tran,omitempty"`
	Vendor     string      `json:"vendor,omitempty"`
	Model      string      `json:"model,omitempty"`
	Serial     string      `json:"serial,omitempty"`
	Mountpoint *string     `json:"mountpoint,omitempty"`
	FSType     string      `json:"fstype,omitempty"`
	RM         *bool       `json:"rm,omitempty"`
	Children   []rawDevice `json:"children,omitempty"`
}

// Device is the normalized block device we expose upstream (pre-filtering)
type Device struct {
	Name        string
	Path        string
	SizeBytes   uint64
	Model       string
	Serial      string
	Rota        *bool
	Removable   *bool
	Type        string
	FSType      string
	BtrfsMember bool
	LUKS        bool
	Warnings    []string
}
