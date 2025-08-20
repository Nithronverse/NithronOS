package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestUpdatesPlan_NonDebianOrNoApt_ReturnsEmptyNote(t *testing.T) {
	body := UpdatesPlanRequest{}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/updates/plan", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	// Force apt-get lookup to fail by clearing PATH
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	handleUpdatesPlan(rr, req)

	if runtime.GOOS == "windows" {
		if rr.Code != http.StatusNotImplemented {
			t.Fatalf("expected 501 on windows, got %d", rr.Code)
		}
		return
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp UpdatesPlanResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Updates) != 0 {
		t.Fatalf("expected no updates, got %d", len(resp.Updates))
	}
	if resp.Note == "" {
		t.Fatalf("expected a note explaining apt missing")
	}
}

func TestParseAptSimulateUpgrade_ParsesLines(t *testing.T) {
	sample := "Inst nosd [0.1.0] (0.2.0 stable [amd64])\n" +
		"Inst nos-agent [0.1.0] (0.2.1 stable [amd64])\n" +
		"Conf nos-web (1.0.0 stable [all])\n" // Conf lines should be ignored
	out := parseAptSimulateUpgrade([]byte(sample))
	if len(out) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(out))
	}
	if out[0].Name != "nosd" || out[0].Current != "0.1.0" || out[0].Candidate != "0.2.0" || out[0].Arch != "amd64" {
		t.Fatalf("unexpected first entry: %+v", out[0])
	}
	if out[1].Name != "nos-agent" || out[1].Candidate != "0.2.1" {
		t.Fatalf("unexpected second entry: %+v", out[1])
	}
	// ensure repo parsed
	if strings.TrimSpace(out[0].Repo) == "" {
		t.Fatalf("expected repo field to be set")
	}
}
