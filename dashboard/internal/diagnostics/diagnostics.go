package diagnostics

import (
	"fmt"
	"os"
	"strings"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/system"
)

const (
	docsDashboard       = "https://github.com/alireza787b/mavlink-anywhere/blob/main/docs/DASHBOARD.md"
	docsTroubleshooting = "https://github.com/alireza787b/mavlink-anywhere/blob/main/docs/TROUBLESHOOTING.md"
	docsUART            = "https://github.com/alireza787b/mavlink-anywhere/blob/main/docs/UART-SETUP.md"
)

type Warning struct {
	Level  string `json:"level"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Hint   string `json:"hint,omitempty"`
	DocURL string `json:"docUrl,omitempty"`
}

type Snapshot struct {
	Warnings []Warning         `json:"warnings"`
	Docs     map[string]string `json:"docs"`
}

func Collect(pc *config.ParsedConfig, svc system.ServiceStatus, board system.BoardInfo, firewall system.FirewallInfo, parseErr error) Snapshot {
	snapshot := Snapshot{
		Warnings: []Warning{},
		Docs: map[string]string{
			"dashboard":       docsDashboard,
			"troubleshooting": docsTroubleshooting,
			"uart":            docsUART,
		},
	}

	if parseErr != nil {
		snapshot.Warnings = append(snapshot.Warnings, Warning{
			Level:  "critical",
			Code:   "config_parse_failed",
			Title:  "Config could not be parsed",
			Detail: parseErr.Error(),
			Hint:   "Restore a valid mavlink-router configuration before editing endpoints in the dashboard.",
			DocURL: docsTroubleshooting,
		})
		return snapshot
	}

	if svc.State != "running" {
		snapshot.Warnings = append(snapshot.Warnings, Warning{
			Level:  "critical",
			Code:   "service_not_running",
			Title:  "mavlink-router is not running",
			Detail: "The router service must be running before telemetry can reach MAVSDK, mavlink2rest, or QGroundControl.",
			Hint:   "Start or restart the service, then refresh the diagnostics panel.",
			DocURL: docsTroubleshooting,
		})
	}

	if pc.General.TcpServerPort <= 0 {
		snapshot.Warnings = append(snapshot.Warnings, Warning{
			Level:  "warning",
			Code:   "tcp_server_disabled",
			Title:  "TCP server is disabled",
			Detail: "The dashboard MAVLink probe and multi-client TCP workflows rely on the mavlink-router TCP server.",
			Hint:   "Set TcpServerPort back to 5760 unless you intentionally disabled it.",
			DocURL: docsDashboard,
		})
	}

	input, hasInput := findInput(pc.Endpoints)
	if !hasInput {
		snapshot.Warnings = append(snapshot.Warnings, Warning{
			Level:  "critical",
			Code:   "no_input_endpoint",
			Title:  "No MAVLink input is configured",
			Detail: "mavlink-router has no UART or UDP input endpoint, so there is no source stream to distribute.",
			Hint:   "Configure either a UART input or a UDP server input.",
			DocURL: docsTroubleshooting,
		})
	} else if input.Type == "UartEndpoint" {
		if _, err := os.Stat(input.Device); err != nil {
			snapshot.Warnings = append(snapshot.Warnings, Warning{
				Level:  "critical",
				Code:   "uart_device_missing",
				Title:  "UART input device is unavailable",
				Detail: fmt.Sprintf("The configured device %s does not exist or is not accessible.", input.Device),
				Hint:   "Check cabling, serial overlays, and Linux UART setup before retrying.",
				DocURL: docsUART,
			})
		}
		if board.BoardType == "raspberry_pi" && board.SerialConsole == "enabled" {
			snapshot.Warnings = append(snapshot.Warnings, Warning{
				Level:  "warning",
				Code:   "serial_console_enabled",
				Title:  "Serial console is still enabled",
				Detail: "The Linux serial console can compete with the flight controller for the same UART.",
				Hint:   "Disable the serial console and reboot before using GPIO UART for MAVLink.",
				DocURL: docsUART,
			})
		}
	}

	if conflict := detectServerBindConflict(pc.Endpoints); conflict != nil {
		snapshot.Warnings = append(snapshot.Warnings, *conflict)
	}

	hasGCSListen := false
	hasExplicitGCSPush14550 := false
	for _, ep := range pc.Endpoints {
		if ep.Type != "UdpEndpoint" || !ep.Enabled {
			continue
		}
		if strings.EqualFold(ep.Mode, "server") && ep.Port == 14550 {
			hasGCSListen = true
		}
		if strings.EqualFold(ep.Mode, "normal") && ep.Port == 14550 && !isLoopback(ep.Address) {
			hasExplicitGCSPush14550 = true
		}
	}

	if hasGCSListen && firewall.Status == "active" && !system.CheckPortOpen(14550) {
		snapshot.Warnings = append(snapshot.Warnings, Warning{
			Level:  "warning",
			Code:   "firewall_blocks_gcs_listen",
			Title:  "Firewall may block ad-hoc GCS connections",
			Detail: "A server-mode GCS listener exists on 14550/udp, but that port is not open in the active firewall.",
			Hint:   "Open 14550/udp or keep QGC on localhost/SSH tunnel only.",
			DocURL: docsTroubleshooting,
		})
	}

	if hasGCSListen && hasExplicitGCSPush14550 {
		snapshot.Warnings = append(snapshot.Warnings, Warning{
			Level:  "info",
			Code:   "mixed_gcs_patterns",
			Title:  "Server-mode and explicit GCS push coexist",
			Detail: "This is valid and does not cause a local bind conflict. However, the same remote GCS should not consume both paths simultaneously or it may see duplicate telemetry.",
			Hint:   "Use gcs_listen for ad-hoc field access, or use explicit push endpoints for deterministic delivery.",
			DocURL: docsDashboard,
		})
	}

	return snapshot
}

func findInput(eps []endpoints.Endpoint) (endpoints.Endpoint, bool) {
	for _, ep := range eps {
		if ep.Type == "UartEndpoint" {
			return ep, true
		}
		if ep.Type == "UdpEndpoint" && strings.EqualFold(ep.Mode, "server") && ep.Name == "input" {
			return ep, true
		}
	}
	return endpoints.Endpoint{}, false
}

func detectServerBindConflict(eps []endpoints.Endpoint) *Warning {
	binds := map[string]string{}
	for _, ep := range eps {
		if ep.Type != "UdpEndpoint" || !ep.Enabled || !strings.EqualFold(ep.Mode, "server") {
			continue
		}
		addr := ep.Address
		if addr == "" {
			addr = "0.0.0.0"
		}
		key := fmt.Sprintf("%s:%d", addr, ep.Port)
		if existing, ok := binds[key]; ok {
			return &Warning{
				Level:  "critical",
				Code:   "duplicate_server_bind",
				Title:  "Two server-mode endpoints bind the same port",
				Detail: fmt.Sprintf("Both %q and %q want to listen on %s.", existing, ep.Name, key),
				Hint:   "Keep only one UDP server endpoint per local address/port.",
				DocURL: docsTroubleshooting,
			}
		}
		binds[key] = ep.Name
	}
	return nil
}

func isLoopback(addr string) bool {
	return addr == "127.0.0.1" || addr == "localhost"
}
