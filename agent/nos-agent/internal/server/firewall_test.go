package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFirewallApply_RejectsTooLarge(t *testing.T) {
	payload := map[string]any{
		"ruleset_text": strings.Repeat("a", 200*1024+1),
		"persist":      false,
	}
	b, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/firewall/apply", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handleFirewallApply(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

func TestFirewallApply_RejectsForbiddenChars(t *testing.T) {
	for _, bad := range []string{"table inet filter { `backtick` }", "table inet filter { $(whoami) }"} {
		payload := map[string]any{
			"ruleset_text": bad,
			"persist":      false,
		}
		b, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/firewall/apply", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handleFirewallApply(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for input %q, got %d", bad, rr.Code)
		}
	}
}
