package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

const testRouterConfig = `[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=/dev/serial0
Baud=57600
`

const validFleetBaselineJSON = `{
  "baseline": {
    "kind": "mavlink-anywhere-profile",
    "general": {"tcpServerPort": 5760, "reportStats": false},
    "endpoints": [
      {"name":"gcs_vpn","type":"UdpEndpoint","mode":"normal","address":"192.0.2.10","port":24550,"category":"gcs","enabled":true}
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

func TestRemoteMutationAllowsExplicitOpenLabMode(t *testing.T) {
	t.Setenv(openMutationsEnv, "true")
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/validate", strings.NewReader(validFleetBaselineJSON))
	request.RemoteAddr = "198.51.100.4:50000"
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected explicit open lab mode to allow remote mutation, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRemoteMutationOpenLabModeDoesNotBypassToken(t *testing.T) {
	t.Setenv(openMutationsEnv, "true")
	t.Setenv(mutationTokenEnv, "test-token")
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/validate", strings.NewReader(validFleetBaselineJSON))
	request.RemoteAddr = "198.51.100.4:50000"
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected token to override open lab mode, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRemoteDashboardRequiresBasicAuthWhenConfigured(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("field-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	t.Setenv(dashboardAuthUserEnv, "operator")
	t.Setenv(dashboardAuthBcryptEnv, string(hash))
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.RemoteAddr = "198.51.100.4:50000"
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("WWW-Authenticate") == "" {
		t.Fatalf("expected basic auth challenge")
	}
}

func TestDashboardAuthTakesPrecedenceOverOpenLabMode(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("field-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	t.Setenv(dashboardAuthUserEnv, "operator")
	t.Setenv(dashboardAuthBcryptEnv, string(hash))
	t.Setenv(openMutationsEnv, "true")
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.RemoteAddr = "198.51.100.4:50000"
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected dashboard auth to remain required, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRemoteDashboardBasicAuthAllowsMutation(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("field-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	t.Setenv(dashboardAuthUserEnv, "operator")
	t.Setenv(dashboardAuthBcryptEnv, string(hash))
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/validate", strings.NewReader(validFleetBaselineJSON))
	request.RemoteAddr = "198.51.100.4:50000"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(dashboardCSRFHeader, "1")
	request.SetBasicAuth("operator", "field-pass")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected basic-authenticated validation to succeed, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"valid":true`) {
		t.Fatalf("expected valid baseline response, got %s", recorder.Body.String())
	}
}

func TestRemoteDashboardBasicAuthMutationRequiresCSRFHeader(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("field-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	t.Setenv(dashboardAuthUserEnv, "operator")
	t.Setenv(dashboardAuthBcryptEnv, string(hash))
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/profiles/validate", strings.NewReader(validFleetBaselineJSON))
	request.RemoteAddr = "198.51.100.4:50000"
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("operator", "field-pass")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected missing CSRF header to be forbidden, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCORSRequiresExplicitAllowedOrigin(t *testing.T) {
	server := NewServer("main.conf", "env", "test")
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/status", nil)
	request.RemoteAddr = "198.51.100.4:50000"
	request.Header.Set("Origin", "https://example.invalid")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected unconfigured CORS preflight to be forbidden, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("unexpected CORS wildcard/header: %s", recorder.Header().Get("Access-Control-Allow-Origin"))
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

func TestLocalEndpointMutationCreatesBackupAndUpdatesEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "main.conf")
	envPath := filepath.Join(dir, "router.env")
	if err := os.WriteFile(configPath, []byte(testRouterConfig), 0644); err != nil {
		t.Fatal(err)
	}
	server := NewServer(configPath, envPath, "test")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints", strings.NewReader(`{
  "name":"qgc","type":"UdpEndpoint","mode":"normal","address":"192.168.1.50","port":14550
}`))
	request.RemoteAddr = "127.0.0.1:40000"
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "[UdpEndpoint qgc]") {
		t.Fatalf("endpoint was not written: %s", written)
	}
	env, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(env), "192.168.1.50:14550") {
		t.Fatalf("env was not synchronized: %s", env)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "backups", "main-*.conf"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one config backup, got %v (%v)", backups, err)
	}
}
