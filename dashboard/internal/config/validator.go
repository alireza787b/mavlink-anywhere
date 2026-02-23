package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

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
	validBauds := []int{9600, 19200, 38400, 57600, 115200, 230400, 460800, 500000, 921600, 1000000}
	for _, v := range validBauds {
		if baud == v {
			return nil
		}
	}
	return fmt.Errorf("invalid baud rate: %d (common values: 57600, 115200, 921600)", baud)
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
