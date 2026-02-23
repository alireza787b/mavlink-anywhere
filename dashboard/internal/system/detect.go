package system

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// BoardInfo holds detected board information.
type BoardInfo struct {
	BoardType   string `json:"boardType"`
	BoardDesc   string `json:"boardDescription"`
	RpiModel    string `json:"rpiModel,omitempty"`
	Arch        string `json:"architecture"`
	Hostname    string `json:"hostname"`
	KernelVer   string `json:"kernelVersion"`
	TotalRAM    string `json:"totalRam"`
	AvailRAM    string `json:"availableRam"`
	UartEnabled string `json:"uartEnabled"`
	SerialConsole string `json:"serialConsole"`
}

// DetectBoard returns information about the current board.
func DetectBoard() BoardInfo {
	info := BoardInfo{
		Arch: runtime.GOARCH,
	}

	info.Hostname, _ = os.Hostname()
	info.KernelVer = readFileFirstLine("/proc/version")
	info.TotalRAM, info.AvailRAM = getMemInfo()
	info.BoardType, info.BoardDesc, info.RpiModel = detectBoardType()
	info.UartEnabled = detectUartEnabled()
	info.SerialConsole = detectSerialConsole()

	return info
}

func detectBoardType() (boardType, desc, rpiModel string) {
	model := readFileFirstLine("/proc/device-tree/model")
	model = strings.TrimRight(model, "\x00")

	if strings.Contains(strings.ToLower(model), "raspberry") {
		boardType = "raspberry_pi"
		rpiModel = classifyRpiModel(model)
		desc = model
		return
	}
	if strings.Contains(strings.ToLower(model), "jetson") {
		return "jetson", model, ""
	}

	// Fallback checks
	if fileExists("/etc/rpi-issue") || dirExists("/opt/vc") {
		return "raspberry_pi", "Raspberry Pi (Unknown Model)", "pi_other"
	}
	if fileExists("/etc/nv_tegra_release") {
		return "jetson", "NVIDIA Jetson", ""
	}

	return "generic_linux", "Generic Linux", ""
}

func classifyRpiModel(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "pi 5"):
		return "pi5"
	case strings.Contains(lower, "pi 4"):
		return "pi4"
	case strings.Contains(lower, "pi 3"):
		return "pi3"
	case strings.Contains(lower, "zero 2"):
		return "pizero2"
	case strings.Contains(lower, "zero"):
		return "pizero"
	default:
		return "pi_other"
	}
}

func getMemInfo() (total, avail string) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return "unknown", "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			total = strings.TrimSpace(strings.TrimPrefix(line, "MemTotal:"))
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			avail = strings.TrimSpace(strings.TrimPrefix(line, "MemAvailable:"))
		}
	}
	return
}

func detectUartEnabled() string {
	configPaths := []string{"/boot/firmware/config.txt", "/boot/config.txt"}
	for _, p := range configPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "enable_uart=1" {
				return "enabled"
			}
		}
	}
	// Check if UART device exists
	for _, dev := range []string{"/dev/serial0", "/dev/ttyS0", "/dev/ttyAMA0"} {
		if fileExists(dev) {
			return "enabled"
		}
	}
	return "unknown"
}

func detectSerialConsole() string {
	cmdlinePaths := []string{"/boot/firmware/cmdline.txt", "/boot/cmdline.txt"}
	for _, p := range cmdlinePaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, "console=serial0") ||
			strings.Contains(content, "console=ttyAMA0") ||
			strings.Contains(content, "console=ttyS0") {
			return "enabled"
		}
		return "disabled"
	}
	return "unknown"
}

func readFileFirstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.SplitN(string(data), "\n", 2)
	if len(lines) > 0 {
		return strings.TrimRight(lines[0], "\x00\n\r")
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// DetectFirewall returns the active firewall type and status.
func DetectFirewall() (fwType, status string) {
	// Try ufw
	if out, err := exec.Command("ufw", "status").CombinedOutput(); err == nil {
		if strings.Contains(string(out), "Status: active") {
			return "ufw", "active"
		}
		return "ufw", "inactive"
	}
	// Try firewalld
	if out, err := exec.Command("firewall-cmd", "--state").CombinedOutput(); err == nil {
		if strings.TrimSpace(string(out)) == "running" {
			return "firewalld", "active"
		}
		return "firewalld", "inactive"
	}
	// Try iptables
	if _, err := exec.LookPath("iptables"); err == nil {
		return "iptables", "available"
	}
	return "none", "none"
}
