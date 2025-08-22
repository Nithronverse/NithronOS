package server

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
)

type smartSummary struct {
	Passed       *bool `json:"passed,omitempty"`
	TemperatureC *int  `json:"temperature_c,omitempty"`
	PowerOnHours *int  `json:"power_on_hours,omitempty"`
	Reallocated  *int  `json:"reallocated,omitempty"`
	MediaErrors  *int  `json:"media_errors,omitempty"`
}

func handleSmartSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	dev := r.URL.Query().Get("device")
	if !validDevicePath(dev) {
		writeErr(w, http.StatusBadRequest, "invalid device")
		return
	}
	sum, err := smartForDevice(dev)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func validDevicePath(p string) bool {
	if p == "" || strings.ContainsAny(p, " \t\n\r\x00") {
		return false
	}
	if !strings.HasPrefix(p, "/dev/") {
		return false
	}
	return true
}

func smartForDevice(dev string) (smartSummary, error) {
	// Prefer standard ATA path first
	out, err := exec.Command("smartctl", "-H", "-A", "-j", dev).CombinedOutput()
	if err != nil {
		// Try NVMe
		out, err = exec.Command("smartctl", "-H", "-A", "-j", "-d", "nvme", dev).CombinedOutput()
	}
	if err != nil {
		return smartSummary{}, err
	}
	return parseSmartctlJSON(out), nil
}

func parseSmartctlJSON(b []byte) smartSummary {
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	var res smartSummary
	if st, ok := m["smart_status"].(map[string]any); ok {
		if p, ok := st["passed"].(bool); ok {
			res.Passed = &p
		}
	}
	if t, ok := m["temperature"].(map[string]any); ok {
		if c, ok := t["current"].(float64); ok {
			v := int(c)
			res.TemperatureC = &v
		}
	}
	if p, ok := m["power_on_time"].(map[string]any); ok {
		if h, ok := p["hours"].(float64); ok {
			v := int(h)
			res.PowerOnHours = &v
		}
	}
	// ATA attributes: find Reallocated_Sector_Ct (id 5)
	if ata, ok := m["ata_smart_attributes"].(map[string]any); ok {
		if tbl, ok := ata["table"].([]any); ok {
			for _, it := range tbl {
				if row, ok := it.(map[string]any); ok {
					if name, _ := row["name"].(string); strings.EqualFold(name, "Reallocated_Sector_Ct") {
						if raw, ok := row["raw"].(map[string]any); ok {
							if val, ok := raw["value"].(float64); ok {
								v := int(val)
								res.Reallocated = &v
							}
						}
					}
				}
			}
		}
	}
	// NVMe
	if nvme, ok := m["nvme_smart_health_information_log"].(map[string]any); ok {
		if me, ok := nvme["media_errors"].(float64); ok {
			v := int(me)
			res.MediaErrors = &v
		}
		// Temperature may also be under this struct depending on drive
		if res.TemperatureC == nil {
			if tc, ok := nvme["temperature"].(float64); ok {
				v := int(tc)
				res.TemperatureC = &v
			}
		}
	}
	return res
}
