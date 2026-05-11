package config

import (
	"path/filepath"
	"testing"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

func TestValidateEndpointTopologyTreatsWildcardAsConflictingBind(t *testing.T) {
	existing := []endpoints.Endpoint{
		{
			Name:    "gcs_any",
			Type:    "UdpEndpoint",
			Mode:    "server",
			Address: "0.0.0.0",
			Port:    14550,
			Enabled: true,
		},
	}
	candidate := endpoints.Endpoint{
		Name:    "gcs_loopback",
		Type:    "UdpEndpoint",
		Mode:    "server",
		Address: "127.0.0.1",
		Port:    14550,
		Enabled: true,
	}

	if err := ValidateEndpointTopology(existing, candidate, ""); err == nil {
		t.Fatalf("expected wildcard bind to conflict with loopback bind on the same port")
	}
}

func TestWriteConfigAndEnvRollsBackConfigWhenEnvWriteFails(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "main.conf")
	envPath := filepath.Join(dir, "missing", "mavlink-router")
	original := "[General]\nTcpServerPort=5760\n\n[UdpEndpoint mavsdk]\nMode=Normal\nAddress=127.0.0.1\nPort=14540\n"
	if err := WriteRawConfig(configPath, filepath.Join(dir, "mavlink-router"), original); err != nil {
		t.Fatalf("initial WriteRawConfig failed: %v", err)
	}

	next := &ParsedConfig{
		General: GeneralSection{TcpServerPort: 5770, ReportStats: true},
		Endpoints: []endpoints.Endpoint{
			{Name: "mavsdk", Type: "UdpEndpoint", Mode: "normal", Address: "127.0.0.1", Port: 14541, Enabled: true},
		},
	}
	if err := WriteConfigAndEnv(configPath, envPath, next); err == nil {
		t.Fatalf("expected env write to fail")
	}
	parsed, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile failed: %v", err)
	}
	if parsed.General.TcpServerPort != 5760 {
		t.Fatalf("expected config rollback to preserve tcp port 5760, got %d", parsed.General.TcpServerPort)
	}
}
