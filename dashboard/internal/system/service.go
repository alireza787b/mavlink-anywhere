package system

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const serviceName = "mavlink-router"

// ServiceStatus holds the current state of the mavlink-router service.
type ServiceStatus struct {
	State     string `json:"state"`     // "running", "stopped", "not_installed"
	Enabled   bool   `json:"enabled"`   // starts at boot
	Uptime    string `json:"uptime"`
	StartedAt string `json:"startedAt"`
}

// GetServiceStatus returns the current status of mavlink-router.
func GetServiceStatus() ServiceStatus {
	ss := ServiceStatus{}

	// Check if active
	if err := exec.Command("systemctl", "is-active", "--quiet", serviceName).Run(); err == nil {
		ss.State = "running"
	} else {
		// Check if unit exists
		out, _ := exec.Command("systemctl", "list-unit-files", serviceName+".service").CombinedOutput()
		if strings.Contains(string(out), serviceName) {
			ss.State = "stopped"
		} else {
			ss.State = "not_installed"
			return ss
		}
	}

	// Check if enabled
	if err := exec.Command("systemctl", "is-enabled", "--quiet", serviceName).Run(); err == nil {
		ss.Enabled = true
	}

	// Get uptime
	if ss.State == "running" {
		out, err := exec.Command("systemctl", "show", serviceName, "--property=ActiveEnterTimestamp").CombinedOutput()
		if err == nil {
			ts := strings.TrimPrefix(strings.TrimSpace(string(out)), "ActiveEnterTimestamp=")
			if ts != "" {
				if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", ts); err == nil {
					ss.StartedAt = t.Format(time.RFC3339)
					ss.Uptime = formatDuration(time.Since(t))
				}
			}
		}
	}

	return ss
}

// RestartService restarts mavlink-router via systemctl.
func RestartService() error {
	return exec.Command("systemctl", "restart", serviceName).Run()
}

// StopService stops mavlink-router.
func StopService() error {
	return exec.Command("systemctl", "stop", serviceName).Run()
}

// StartService starts mavlink-router.
func StartService() error {
	return exec.Command("systemctl", "start", serviceName).Run()
}

// OpenFirewallPort opens a UDP port using the detected firewall tool.
func OpenFirewallPort(port int) error {
	fwType, _ := DetectFirewall()
	switch fwType {
	case "ufw":
		return exec.Command("ufw", "allow", fmt.Sprintf("%d/udp", port)).Run()
	case "firewalld":
		return exec.Command("firewall-cmd", "--add-port", fmt.Sprintf("%d/udp", port), "--permanent").Run()
	case "iptables":
		return exec.Command("iptables", "-A", "INPUT", "-p", "udp", "--dport", fmt.Sprintf("%d", port), "-j", "ACCEPT").Run()
	default:
		return nil // No firewall
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 24 {
		days := h / 24
		hours := h % 24
		return fmt.Sprintf("%dd %dh %dm", days, hours, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
