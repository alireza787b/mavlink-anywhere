package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

// ValidateEndpoint validates one endpoint before it reaches the config file.
func ValidateEndpoint(ep endpoints.Endpoint) error {
	if err := ValidateEndpointName(ep.Name); err != nil {
		return err
	}
	switch ep.Type {
	case "UartEndpoint":
		if err := ValidateUartDevice(ep.Device); err != nil {
			return err
		}
		return ValidateBaud(ep.Baud)
	case "UdpEndpoint":
		if err := ValidateEndpointMode(ep.Mode); err != nil {
			return err
		}
		if err := ValidateIP(ep.Address); err != nil {
			return err
		}
		return ValidatePort(ep.Port)
	case "TcpEndpoint":
		if err := ValidateIP(ep.Address); err != nil {
			return err
		}
		return ValidatePort(ep.Port)
	default:
		return fmt.Errorf("unsupported endpoint type: %s", ep.Type)
	}
}

// ValidateParsedConfig checks the complete effective config, including names,
// endpoint fields, duplicate names and conflicting server-mode UDP binds.
func ValidateParsedConfig(pc *ParsedConfig) error {
	if pc == nil {
		return fmt.Errorf("config is empty")
	}
	if err := ValidatePort(pc.General.TcpServerPort); err != nil {
		return fmt.Errorf("TCP server: %w", err)
	}
	if len(pc.Endpoints) == 0 {
		return fmt.Errorf("config must contain at least one endpoint")
	}
	names := map[string]bool{}
	validated := make([]endpoints.Endpoint, 0, len(pc.Endpoints))
	active := 0
	for _, ep := range pc.Endpoints {
		if names[ep.Name] {
			return fmt.Errorf("duplicate endpoint name: %s", ep.Name)
		}
		names[ep.Name] = true
		if !ep.Enabled {
			continue
		}
		active++
		if err := ValidateEndpoint(ep); err != nil {
			return fmt.Errorf("endpoint %s: %w", ep.Name, err)
		}
		if err := ValidateEndpointTopology(validated, ep, ""); err != nil {
			return err
		}
		validated = append(validated, ep)
	}
	if active == 0 {
		return fmt.Errorf("config must contain at least one enabled endpoint")
	}
	return nil
}

// ValidateIP checks if a string is a valid IPv4 address.
func ValidateIP(ip string) error {
	if ip == "" {
		return fmt.Errorf("IP address is required")
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	if parsed.To4() == nil {
		return fmt.Errorf("only IPv4 addresses supported: %s", ip)
	}
	return nil
}

// ValidatePort checks if a port number is valid.
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// ValidateEndpointMode ensures endpoint mode is supported.
func ValidateEndpointMode(mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "normal", "server":
		return nil
	default:
		return fmt.Errorf("unsupported endpoint mode: %s", mode)
	}
}

// ValidateEndpointName checks endpoint name validity.
func ValidateEndpointName(name string) error {
	if name == "" {
		return fmt.Errorf("endpoint name is required")
	}
	if len(name) > 32 {
		return fmt.Errorf("endpoint name must be 32 characters or less")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return fmt.Errorf("endpoint name may only contain letters, digits, underscore, and hyphen")
		}
	}
	return nil
}

// ValidateBaud checks if a baud rate is valid.
func ValidateBaud(baud int) error {
	if baud < 1200 || baud > 4000000 {
		return fmt.Errorf("invalid baud rate: %d (common values: 57600, 115200, 921600)", baud)
	}
	return nil
}

// ValidateUartDevice checks basic UART device path validity.
func ValidateUartDevice(device string) error {
	if device == "" {
		return fmt.Errorf("UART device path is required")
	}
	if !strings.HasPrefix(device, "/dev/") {
		return fmt.Errorf("UART device must start with /dev/")
	}
	return nil
}

// ParseEndpointString parses "IP:PORT" format.
func ParseEndpointString(s string) (string, int, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("expected IP:PORT format, got %q", s)
	}
	if err := ValidateIP(parts[0]); err != nil {
		return "", 0, err
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port: %s", parts[1])
	}
	if err := ValidatePort(port); err != nil {
		return "", 0, err
	}
	return parts[0], port, nil
}
