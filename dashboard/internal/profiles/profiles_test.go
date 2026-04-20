package profiles

import (
	"path/filepath"
	"testing"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

func TestExportAndPreviewRoundTrip(t *testing.T) {
	current := sampleCurrentConfig()

	profile, err := Export(current, "v3.0.6")
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	preview, err := PreviewApply(current, profile, ModeReplace)
	if err != nil {
		t.Fatalf("PreviewApply failed: %v", err)
	}

	if !preview.RestartRequired {
		t.Fatalf("expected restart to be required")
	}
	if preview.RebootRequired {
		t.Fatalf("did not expect reboot to be required")
	}
	if len(preview.Changes.Added) != 0 || len(preview.Changes.Removed) != 0 || len(preview.Changes.Updated) != 0 {
		t.Fatalf("expected no config deltas for round-trip preview, got %+v", preview.Changes)
	}
}

func TestApplyReplaceWritesConfigEnvAndBackup(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "main.conf")
	envPath := filepath.Join(dir, "mavlink-router")

	current := sampleCurrentConfig()
	if err := config.WriteConfigAndEnv(configPath, envPath, current); err != nil {
		t.Fatalf("WriteConfigAndEnv failed: %v", err)
	}

	nextProfile := Profile{
		SchemaVersion: SchemaVersion,
		Kind:          Kind,
		Metadata: Metadata{
			ProfileName: "replace-profile",
			ExportedAt:  "2026-04-20T00:00:00Z",
			ExportedBy:  "test",
		},
		General: config.GeneralSection{
			TcpServerPort: 0,
			ReportStats:   false,
		},
		Endpoints: []endpoints.Endpoint{
			{
				Name:     "uart",
				Type:     "UartEndpoint",
				Device:   "/dev/ttyAMA0",
				Baud:     921600,
				Category: "input",
				Enabled:  true,
			},
			{
				Name:     "mavsdk",
				Type:     "UdpEndpoint",
				Mode:     "normal",
				Address:  "127.0.0.1",
				Port:     14540,
				Category: "local",
				Enabled:  true,
			},
		},
	}

	backup, target, err := Apply(configPath, envPath, current, nextProfile, ModeReplace)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if backup.ConfigBackup == "" || backup.MetadataPath == "" {
		t.Fatalf("expected backup info to be populated: %+v", backup)
	}
	if target.General.TcpServerPort != 0 {
		t.Fatalf("expected tcp server port 0, got %d", target.General.TcpServerPort)
	}

	parsed, err := config.ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile failed: %v", err)
	}
	if parsed.General.TcpServerPort != 0 {
		t.Fatalf("expected written tcp server port 0, got %d", parsed.General.TcpServerPort)
	}
	env, err := config.ParseEnvFile(envPath)
	if err != nil {
		t.Fatalf("ParseEnvFile failed: %v", err)
	}
	if env.InputType != "uart" || env.UartDevice != "/dev/ttyAMA0" || env.UartBaud != "921600" {
		t.Fatalf("env file did not reflect UART input: %+v", env)
	}
	if env.UdpEndpoints != "127.0.0.1:14540" {
		t.Fatalf("unexpected UDP_ENDPOINTS: %q", env.UdpEndpoints)
	}
}

func TestApplyMergePreservesUnnamedEndpoints(t *testing.T) {
	current := sampleCurrentConfig()
	profile := Profile{
		SchemaVersion: SchemaVersion,
		Kind:          Kind,
		Metadata: Metadata{
			ProfileName: "merge-profile",
			ExportedAt:  "2026-04-20T00:00:00Z",
			ExportedBy:  "test",
		},
		General: current.General,
		Endpoints: []endpoints.Endpoint{
			{
				Name:     "uart",
				Type:     "UartEndpoint",
				Device:   "/dev/ttyS0",
				Baud:     921600,
				Category: "input",
				Enabled:  true,
			},
			{
				Name:     "mavsdk",
				Type:     "UdpEndpoint",
				Mode:     "normal",
				Address:  "127.0.0.1",
				Port:     14541,
				Category: "local",
				Enabled:  true,
			},
			{
				Name:     "gcs_vpn",
				Type:     "UdpEndpoint",
				Mode:     "normal",
				Address:  "100.82.10.10",
				Port:     24550,
				Category: "gcs",
				Enabled:  true,
			},
		},
	}

	preview, err := PreviewApply(current, profile, ModeMerge)
	if err != nil {
		t.Fatalf("PreviewApply failed: %v", err)
	}
	if len(preview.Changes.Added) != 1 || preview.Changes.Added[0] != "gcs_vpn" {
		t.Fatalf("expected gcs_vpn to be added, got %+v", preview.Changes)
	}
	if len(preview.Changes.Updated) != 1 || preview.Changes.Updated[0] != "mavsdk" {
		t.Fatalf("expected mavsdk to be updated, got %+v", preview.Changes)
	}
	if len(preview.Changes.Preserved) != 2 {
		t.Fatalf("expected two preserved endpoints, got %+v", preview.Changes)
	}
}

func TestRestoreLatestRestoresPreviousConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "main.conf")
	envPath := filepath.Join(dir, "mavlink-router")

	current := sampleCurrentConfig()
	if err := config.WriteConfigAndEnv(configPath, envPath, current); err != nil {
		t.Fatalf("WriteConfigAndEnv failed: %v", err)
	}

	profile := Profile{
		SchemaVersion: SchemaVersion,
		Kind:          Kind,
		Metadata: Metadata{
			ProfileName: "restore-profile",
			ExportedAt:  "2026-04-20T00:00:00Z",
			ExportedBy:  "test",
		},
		General: current.General,
		Endpoints: []endpoints.Endpoint{
			{
				Name:     "uart",
				Type:     "UartEndpoint",
				Device:   "/dev/ttyAMA0",
				Baud:     921600,
				Category: "input",
				Enabled:  true,
			},
			{
				Name:     "mavsdk",
				Type:     "UdpEndpoint",
				Mode:     "normal",
				Address:  "127.0.0.1",
				Port:     14541,
				Category: "local",
				Enabled:  true,
			},
		},
	}

	if _, _, err := Apply(configPath, envPath, current, profile, ModeReplace); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if _, err := RestoreLatest(configPath, envPath); err != nil {
		t.Fatalf("RestoreLatest failed: %v", err)
	}

	restored, err := config.ParseConfigFile(configPath)
	if err != nil {
		t.Fatalf("ParseConfigFile failed: %v", err)
	}
	if restored.General != current.General {
		t.Fatalf("expected general config to be restored, got %+v", restored.General)
	}
	if len(restored.Endpoints) != len(current.Endpoints) {
		t.Fatalf("expected %d endpoints after restore, got %d", len(current.Endpoints), len(restored.Endpoints))
	}
}

func TestValidateRejectsDuplicateServerBind(t *testing.T) {
	profile := Profile{
		SchemaVersion: SchemaVersion,
		Kind:          Kind,
		Metadata: Metadata{
			ProfileName: "invalid-profile",
			ExportedAt:  "2026-04-20T00:00:00Z",
			ExportedBy:  "test",
		},
		General: config.GeneralSection{
			TcpServerPort: 5760,
		},
		Endpoints: []endpoints.Endpoint{
			{
				Name:     "input",
				Type:     "UdpEndpoint",
				Mode:     "server",
				Address:  "0.0.0.0",
				Port:     14550,
				Category: "input",
				Enabled:  true,
			},
			{
				Name:     "gcs_listen",
				Type:     "UdpEndpoint",
				Mode:     "server",
				Address:  "0.0.0.0",
				Port:     14550,
				Category: "gcs",
				Enabled:  true,
			},
		},
	}

	if err := Validate(profile); err == nil {
		t.Fatalf("expected duplicate server bind validation error")
	}
}

func sampleCurrentConfig() *config.ParsedConfig {
	return &config.ParsedConfig{
		General: config.GeneralSection{
			TcpServerPort: 5760,
			ReportStats:   true,
		},
		Endpoints: []endpoints.Endpoint{
			{
				Name:     "uart",
				Type:     "UartEndpoint",
				Device:   "/dev/ttyS0",
				Baud:     57600,
				Category: "input",
				Enabled:  true,
			},
			{
				Name:     "gcs_listen",
				Type:     "UdpEndpoint",
				Mode:     "server",
				Address:  "0.0.0.0",
				Port:     14550,
				Category: "gcs",
				Enabled:  true,
			},
			{
				Name:     "mavsdk",
				Type:     "UdpEndpoint",
				Mode:     "normal",
				Address:  "127.0.0.1",
				Port:     14540,
				Category: "local",
				Enabled:  true,
			},
			{
				Name:     "gcs_home",
				Type:     "UdpEndpoint",
				Mode:     "normal",
				Address:  "192.168.1.10",
				Port:     24550,
				Category: "gcs",
				Enabled:  true,
			},
		},
	}
}
