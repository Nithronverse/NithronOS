package server

import (
	"os"
	"testing"
)

func TestParseSmartctlJSON_ATA(t *testing.T) {
	b, err := os.ReadFile("testdata/smart_ata.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sum := parseSmartctlJSON(b)
	if sum.Passed == nil || *sum.Passed != true {
		t.Fatalf("expected passed=true: %+v", sum)
	}
	if sum.TemperatureC == nil || *sum.TemperatureC != 35 {
		t.Fatalf("expected temp 35C, got %+v", sum)
	}
	if sum.PowerOnHours == nil || *sum.PowerOnHours != 1234 {
		t.Fatalf("expected poh 1234, got %+v", sum)
	}
	if sum.Reallocated == nil || *sum.Reallocated != 0 {
		t.Fatalf("expected reallocated 0, got %+v", sum)
	}
}

func TestParseSmartctlJSON_NVMe(t *testing.T) {
	b, err := os.ReadFile("testdata/smart_nvme.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	sum := parseSmartctlJSON(b)
	if sum.Passed == nil || *sum.Passed != true {
		t.Fatalf("expected passed=true: %+v", sum)
	}
	if sum.TemperatureC == nil || *sum.TemperatureC != 45 {
		t.Fatalf("expected temp 45C, got %+v", sum)
	}
	if sum.MediaErrors == nil || *sum.MediaErrors != 1 {
		t.Fatalf("expected media_errors 1, got %+v", sum)
	}
}
