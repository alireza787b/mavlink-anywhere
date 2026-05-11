package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const validFleetBaselineJSON = `{
  "baseline": {
    "kind": "mavlink-anywhere-profile",
    "general": {"tcpServerPort": 5760, "reportStats": false},
    "endpoints": [
      {"name":"gcs_vpn","type":"UdpEndpoint","mode":"normal","address":"100.64.0.10","port":24550,"category":"gcs","enabled":true}
    ]
  }
}`

func TestRemoteMutationRequiresToken(t *testing.T) {
	t.Setenv(mutationTokenEnv, "")
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/validate", strings.NewReader(validFleetBaselineJSON))
	request.RemoteAddr = "198.51.100.4:50000"
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected remote mutation without token to be forbidden, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRemoteMutationAcceptsConfiguredToken(t *testing.T) {
	t.Setenv(mutationTokenEnv, "test-token")
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/validate", strings.NewReader(validFleetBaselineJSON))
	request.RemoteAddr = "198.51.100.4:50000"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer test-token")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected token-authenticated validation to succeed, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"valid":true`) {
		t.Fatalf("expected valid baseline response, got %s", recorder.Body.String())
	}
}

func TestFleetApplyAcceptsConfirmationTokenAlias(t *testing.T) {
	var req fleetProfileApplyRequest
	raw := `{"dry_run_id":"mla-example","confirmation":{"acknowledged_risks":true,"confirmation_token":"dry-run-token"}}`
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if req.Confirmation.ConfirmationToken != "dry-run-token" {
		t.Fatalf("expected confirmation_token alias to be decoded")
	}
}
