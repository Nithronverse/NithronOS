package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSMBUserCreate_InvalidUsername(t *testing.T) {
	body := SMBUserCreateRequest{Username: "bad name"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/smb/user-create", bytes.NewReader(b))
	w := httptest.NewRecorder()
	handleSMBUserCreate(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSMBUsersList_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/smb/users", nil)
	w := httptest.NewRecorder()
	handleSMBUsersList(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
