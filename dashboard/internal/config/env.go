package config

import (
	"bufio"
	"os"
	"strings"
)

// EnvConfig holds values from /etc/default/mavlink-router.
type EnvConfig struct {
	InputType    string `json:"inputType"`
	UartDevice   string `json:"uartDevice"`
	UartBaud     string `json:"uartBaud"`
	InputAddress string `json:"inputAddress"`
	InputPort    string `json:"inputPort"`
	UdpEndpoints string `json:"udpEndpoints"`
}

// ParseEnvFile reads the mavlink-router environment file.
func ParseEnvFile(path string) (*EnvConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return &EnvConfig{}, nil // Return empty if not found
	}
	defer f.Close()

	env := &EnvConfig{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.Trim(strings.TrimSpace(line[eqIdx+1:]), "\"")

		switch key {
		case "INPUT_TYPE":
			env.InputType = val
		case "UART_DEVICE":
			env.UartDevice = val
		case "UART_BAUD":
			env.UartBaud = val
		case "INPUT_ADDRESS":
			env.InputAddress = val
		case "INPUT_PORT":
			env.InputPort = val
		case "UDP_ENDPOINTS":
			env.UdpEndpoints = val
		}
	}
	return env, nil
}
