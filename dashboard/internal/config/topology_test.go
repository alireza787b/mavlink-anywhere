package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestToggleEndpointPreservesDisabledEndpointForReenable(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "main.conf")
	raw := `[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=/dev/serial0
Baud=57600

[UdpEndpoint mavsdk]
Mode=normal
Address=127.0.0.1
Port=14540
`
	if err := os.WriteFile(configPath, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ToggleEndpoint(configPath, "mavsdk", false); err != nil {
		t.Fatal(err)
	}
	disabled, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled.Endpoints) != 2 || disabled.Endpoints[1].Enabled {
		t.Fatalf("expected disabled endpoint to remain parseable: %#v", disabled.Endpoints)
	}
	if disabled.Endpoints[1].Address != "127.0.0.1" || disabled.Endpoints[1].Port != 14540 {
		t.Fatalf("disabled endpoint fields were not preserved: %#v", disabled.Endpoints[1])
	}
	if err := ToggleEndpoint(configPath, "mavsdk", true); err != nil {
		t.Fatal(err)
	}
	enabled, err := ParseConfigFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled.Endpoints[1].Enabled {
		t.Fatal("expected endpoint to be enabled again")
	}
}

func TestWriteRawConfigRejectsInvalidEndpoint(t *testing.T) {
	dir := t.TempDir()
	raw := `[General]
TcpServerPort=5760

[UdpEndpoint broken]
Mode=normal
Address=999.999.999.999
Port=70000
`
	err := WriteRawConfig(filepath.Join(dir, "main.conf"), filepath.Join(dir, "router.env"), raw)
	if err == nil {
		t.Fatal("expected invalid raw config to be rejected")
	}
}

func TestWriteRawConfigPreservesCommentsAndUnknownSections(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "main.conf")
	envPath := filepath.Join(dir, "router.env")
	raw := `# operator note
[General]
TcpServerPort=5760

[Log telemetry]
Type=bin

[UartEndpoint uart]
Device=/dev/serial0
Baud=420000
`
	if err := WriteRawConfig(configPath, envPath, raw); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != raw {
		t.Fatalf("raw config was rewritten:\n%s", written)
	}
}

func TestEndpointMutationsPreserveAdvancedRawSections(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "main.conf")
	raw := `# keep this note
[General]
TcpServerPort=5760

[Log telemetry]
Type=bin

[UartEndpoint uart]
Device=/dev/serial0
Baud=57600

# keep this route note
[UdpEndpoint mavsdk]
Mode=normal
Address=127.0.0.1
Port=14540
`
	if err := os.WriteFile(configPath, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	qgc := endpoints.Endpoint{Name: "qgc", Type: "UdpEndpoint", Mode: "normal", Address: "192.168.1.50", Port: 14550, Enabled: true}
	if err := AddEndpoint(configPath, qgc); err != nil {
		t.Fatal(err)
	}
	qgc.Address = "192.168.1.60"
	if err := UpdateEndpoint(configPath, "qgc", qgc); err != nil {
		t.Fatal(err)
	}
	if err := ToggleEndpoint(configPath, "qgc", false); err != nil {
		t.Fatal(err)
	}
	if err := ToggleEndpoint(configPath, "qgc", true); err != nil {
		t.Fatal(err)
	}
	if err := DeleteEndpoint(configPath, "qgc"); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# keep this note", "[Log telemetry]", "Type=bin", "# keep this route note"} {
		if !strings.Contains(string(written), expected) {
			t.Fatalf("advanced raw content %q was lost:\n%s", expected, written)
		}
	}
}
