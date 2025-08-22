package disks

type SmartSummary struct {
	Healthy      *bool `json:"healthy,omitempty"`
	TempCelsius  *int  `json:"temp_c,omitempty"`
	PowerOnHours *int  `json:"power_on_hours,omitempty"`
	Reallocated  *int  `json:"reallocated_sectors,omitempty"`
	MediaErrors  *int  `json:"media_errors,omitempty"`
}

type Disk struct {
	Name       string        `json:"name"`
	KName      string        `json:"kname,omitempty"`
	Path       string        `json:"path,omitempty"`
	SizeBytes  int64         `json:"size"`
	Rota       *bool         `json:"rota,omitempty"`
	Type       string        `json:"type"`
	Tran       string        `json:"tran,omitempty"`
	Vendor     string        `json:"vendor,omitempty"`
	Model      string        `json:"model,omitempty"`
	Serial     string        `json:"serial,omitempty"`
	Mountpoint *string       `json:"mountpoint,omitempty"`
	FSType     string        `json:"fstype,omitempty"`
	Smart      *SmartSummary `json:"smart,omitempty"`
}
