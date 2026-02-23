package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// FirewallInfo holds firewall status.
type FirewallInfo struct {
	Type      string   `json:"type"`      // "ufw", "firewalld", "iptables", "none"
	Status    string   `json:"status"`    // "active", "inactive", "none"
	OpenPorts []string `json:"openPorts"` // List of open UDP ports
}

// GetFirewallInfo returns current firewall status and open ports.
func GetFirewallInfo() FirewallInfo {
	fwType, status := DetectFirewall()
	info := FirewallInfo{
		Type:   fwType,
		Status: status,
	}

	switch fwType {
	case "ufw":
		out, err := exec.Command("ufw", "status").CombinedOutput()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "/udp") && strings.Contains(line, "ALLOW") {
					parts := strings.Fields(line)
					if len(parts) > 0 {
						info.OpenPorts = append(info.OpenPorts, parts[0])
					}
				}
			}
		}
	case "firewalld":
		out, err := exec.Command("firewall-cmd", "--list-ports").CombinedOutput()
		if err == nil {
			for _, port := range strings.Fields(string(out)) {
				if strings.HasSuffix(port, "/udp") {
					info.OpenPorts = append(info.OpenPorts, port)
				}
			}
		}
	}

	if info.OpenPorts == nil {
		info.OpenPorts = []string{}
	}

	return info
}

// CheckPortOpen checks if a specific UDP port is open in the firewall.
func CheckPortOpen(port int) bool {
	info := GetFirewallInfo()
	if info.Status != "active" {
		return true // No active firewall
	}

	portStr := fmt.Sprintf("%d/udp", port)
	for _, p := range info.OpenPorts {
		if p == portStr {
			return true
		}
	}
	return false
}
