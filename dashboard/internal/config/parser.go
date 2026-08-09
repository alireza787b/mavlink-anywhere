package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

// ParsedConfig represents the full mavlink-router configuration.
type ParsedConfig struct {
	General    GeneralSection       `json:"general"`
	Endpoints  []endpoints.Endpoint `json:"endpoints"`
	Raw        string               `json:"raw"`
	ModifiedAt time.Time            `json:"modifiedAt"`
}

// GeneralSection holds [General] section values.
type GeneralSection struct {
	TcpServerPort int  `json:"tcpServerPort"`
	ReportStats   bool `json:"reportStats"`
}

// ParseConfigFile reads and parses an INI-style mavlink-router config file.
func ParseConfigFile(path string) (*ParsedConfig, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat config: %w", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	return ParseConfigText(string(raw), info.ModTime())
}

// ParseConfigText parses an INI-style mavlink-router config string.
func ParseConfigText(raw string, modifiedAt time.Time) (*ParsedConfig, error) {
	pc := &ParsedConfig{
		General: GeneralSection{
			TcpServerPort: 5760,
		},
		Raw:        raw,
		ModifiedAt: modifiedAt,
	}

	scanner := bufio.NewScanner(strings.NewReader(expandDisabledEndpoints(raw)))

	var currentSection string
	var currentName string
	var currentType string
	props := map[string]string{}
	generalSections := 0

	flushEndpoint := func() {
		if currentType == "" || currentName == "" {
			return
		}
		ep := buildEndpoint(currentType, currentName, props)
		pc.Endpoints = append(pc.Endpoints, ep)
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && !strings.HasSuffix(line, "]") {
			return nil, fmt.Errorf("malformed section header: %s", line)
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Flush previous endpoint
			if currentSection != "General" {
				flushEndpoint()
			}
			props = map[string]string{}

			section := line[1 : len(line)-1]
			parts := strings.SplitN(section, " ", 2)
			currentSection = parts[0]
			currentName = ""
			currentType = ""
			if len(parts) > 1 {
				currentName = parts[1]
			}

			switch {
			case currentSection == "General":
				generalSections++
				if generalSections > 1 {
					return nil, fmt.Errorf("duplicate [General] section")
				}
			case strings.HasPrefix(currentSection, "UdpEndpoint"):
				currentType = "UdpEndpoint"
				if currentName == "" {
					return nil, fmt.Errorf("UdpEndpoint section requires a name")
				}
			case strings.HasPrefix(currentSection, "UartEndpoint"):
				currentType = "UartEndpoint"
				if currentName == "" {
					return nil, fmt.Errorf("UartEndpoint section requires a name")
				}
			case strings.HasPrefix(currentSection, "TcpEndpoint"):
				currentType = "TcpEndpoint"
				if currentName == "" {
					return nil, fmt.Errorf("TcpEndpoint section requires a name")
				}
			}
			continue
		}

		// Key=Value
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			if currentSection == "General" || currentType != "" {
				return nil, fmt.Errorf("invalid setting line: %s", line)
			}
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])

		if currentSection == "General" {
			switch key {
			case "TcpServerPort":
				if v, err := strconv.Atoi(val); err == nil {
					pc.General.TcpServerPort = v
				}
			case "ReportStats":
				pc.General.ReportStats = strings.EqualFold(val, "true")
			}
		} else {
			props[key] = val
		}
	}

	// Flush last section
	flushEndpoint()
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan config: %w", err)
	}
	if generalSections != 1 {
		return nil, fmt.Errorf("config must contain exactly one [General] section")
	}

	return pc, nil
}

const (
	disabledBegin = "# MAVLINK_ANYWHERE_DISABLED_BEGIN "
	disabledEnd   = "# MAVLINK_ANYWHERE_DISABLED_END "
)

// expandDisabledEndpoints turns dashboard-managed commented blocks into normal
// parser input with a private Enabled marker. mavlink-router itself continues
// to ignore the commented block.
func expandDisabledEndpoints(raw string) string {
	var b strings.Builder
	inDisabled := false
	for _, line := range strings.SplitAfter(raw, "\n") {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
		if strings.HasPrefix(trimmed, disabledBegin) {
			inDisabled = true
			continue
		}
		if strings.HasPrefix(trimmed, disabledEnd) {
			inDisabled = false
			continue
		}
		if inDisabled {
			uncommented := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
			if strings.HasPrefix(uncommented, "[") {
				b.WriteString(uncommented)
				b.WriteString("\nMavlinkAnywhereEnabled=false\n")
			} else if uncommented != "" {
				b.WriteString(uncommented)
				b.WriteByte('\n')
			}
			continue
		}
		b.WriteString(line)
	}
	return b.String()
}

func buildEndpoint(epType, name string, props map[string]string) endpoints.Endpoint {
	ep := endpoints.Endpoint{
		Name:    name,
		Type:    epType,
		Enabled: !strings.EqualFold(props["MavlinkAnywhereEnabled"], "false"),
	}

	switch epType {
	case "UartEndpoint":
		ep.Device = props["Device"]
		ep.Baud, _ = strconv.Atoi(props["Baud"])
		ep.Category = "input"
		ep.Description = "Serial connection to flight controller"
		ep.Removable = false

	case "UdpEndpoint":
		ep.Mode = strings.ToLower(props["Mode"])
		if ep.Mode == "" {
			ep.Mode = "normal"
		}
		ep.Address = props["Address"]
		ep.Port, _ = strconv.Atoi(props["Port"])
		ep.Description = endpoints.DescriptionForEndpoint(name, ep.Mode, ep.Address, ep.Port)
		ep.Category = endpoints.CategoryForEndpoint(name, ep.Mode, ep.Address, ep.Port)
		ep.Removable = true

	case "TcpEndpoint":
		ep.Address = props["Address"]
		ep.Port, _ = strconv.Atoi(props["Port"])
		ep.Category = "custom"
		ep.Description = "TCP endpoint"
		ep.Removable = true
	}

	return ep
}

// WriteConfigFile writes a full INI config file preserving the mavlink-router format.
func WriteConfigFile(path string, pc *ParsedConfig) error {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# MAVLink Router Configuration\n"))
	b.WriteString(fmt.Sprintf("# Generated by mavlink-anywhere dashboard at %s\n\n", time.Now().Format(time.RFC3339)))

	// [General]
	b.WriteString("[General]\n")
	b.WriteString(fmt.Sprintf("TcpServerPort=%d\n", pc.General.TcpServerPort))
	b.WriteString(fmt.Sprintf("ReportStats=%s\n", strconv.FormatBool(pc.General.ReportStats)))
	b.WriteString("\n")

	// Endpoints
	for _, ep := range pc.Endpoints {
		if !ep.Enabled {
			b.WriteString(disabledBegin + ep.Name + "\n")
			for _, line := range strings.Split(strings.TrimSuffix(formatEndpoint(ep), "\n"), "\n") {
				if line == "" {
					b.WriteString("#\n")
				} else {
					b.WriteString("# " + line + "\n")
				}
			}
			b.WriteString(disabledEnd + ep.Name + "\n\n")
			continue
		}
		b.WriteString(formatEndpoint(ep))
	}

	return os.WriteFile(path, []byte(b.String()), 0644)
}

func formatEndpoint(ep endpoints.Endpoint) string {
	var b strings.Builder
	switch ep.Type {
	case "UartEndpoint":
		b.WriteString(fmt.Sprintf("[UartEndpoint %s]\nDevice=%s\nBaud=%d\n\n", ep.Name, ep.Device, ep.Baud))
	case "UdpEndpoint":
		b.WriteString(fmt.Sprintf("[UdpEndpoint %s]\nMode=%s\nAddress=%s\nPort=%d\n\n", ep.Name, ep.Mode, ep.Address, ep.Port))
	case "TcpEndpoint":
		b.WriteString(fmt.Sprintf("[TcpEndpoint %s]\nAddress=%s\nPort=%d\n\n", ep.Name, ep.Address, ep.Port))
	}
	return b.String()
}

// WriteConfigAndEnv writes the config file and regenerates the companion env file.
func WriteConfigAndEnv(configPath, envPath string, pc *ParsedConfig) error {
	oldConfig, hadConfig, err := readOptionalFile(configPath)
	if err != nil {
		return err
	}
	oldEnv, hadEnv, err := readOptionalFile(envPath)
	if err != nil {
		return err
	}
	if err := WriteConfigFile(configPath, pc); err != nil {
		return err
	}
	if err := WriteEnvFile(envPath, EnvFromConfig(pc)); err != nil {
		_ = restoreOptionalFile(configPath, oldConfig, hadConfig)
		_ = restoreOptionalFile(envPath, oldEnv, hadEnv)
		return err
	}
	return nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		return raw, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, err
}

func restoreOptionalFile(path string, raw []byte, existed bool) error {
	if existed {
		return os.WriteFile(path, raw, 0644)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// WriteRawConfig validates and writes raw config text, then regenerates the env file.
func WriteRawConfig(configPath, envPath, raw string) error {
	pc, err := ParseConfigText(raw, time.Now())
	if err != nil {
		return err
	}
	if err := ValidateParsedConfig(pc); err != nil {
		return err
	}
	oldConfig, hadConfig, err := readOptionalFile(configPath)
	if err != nil {
		return err
	}
	oldEnv, hadEnv, err := readOptionalFile(envPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(configPath, []byte(raw), 0644); err != nil {
		return err
	}
	if err := WriteEnvFile(envPath, EnvFromConfig(pc)); err != nil {
		_ = restoreOptionalFile(configPath, oldConfig, hadConfig)
		_ = restoreOptionalFile(envPath, oldEnv, hadEnv)
		return err
	}
	return nil
}

// ConfigModTime returns the modification time of the config file.
func ConfigModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// AddEndpoint adds a new endpoint to a parsed config and writes it.
func AddEndpoint(path string, ep endpoints.Endpoint) error {
	pc, err := ParseConfigFile(path)
	if err != nil {
		return err
	}

	// Check for name conflict
	for _, existing := range pc.Endpoints {
		if existing.Name == ep.Name {
			return fmt.Errorf("endpoint %q already exists", ep.Name)
		}
	}
	if err := ValidateEndpointTopology(pc.Endpoints, ep, ""); err != nil {
		return err
	}

	ep.Enabled = true
	updated := strings.TrimRight(pc.Raw, "\r\n") + "\n\n" + formatEndpoint(ep)
	return writeValidatedRawConfig(path, updated)
}

// UpdateEndpoint replaces an endpoint by name.
func UpdateEndpoint(path string, name string, ep endpoints.Endpoint) error {
	pc, err := ParseConfigFile(path)
	if err != nil {
		return err
	}

	found := false
	enabled := true
	for i, existing := range pc.Endpoints {
		if existing.Name == name {
			ep.Type = existing.Type
			if ep.Type == "" {
				ep.Type = "UdpEndpoint"
			}
			if err := ValidateEndpointTopology(pc.Endpoints, ep, name); err != nil {
				return err
			}
			enabled = existing.Enabled
			ep.Enabled = enabled
			pc.Endpoints[i] = ep
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("endpoint %q not found", name)
	}

	rangeInfo, err := findRawEndpointRange(pc.Raw, name)
	if err != nil {
		return err
	}
	replacement := formatEndpoint(ep)
	if !enabled {
		replacement = formatDisabledEndpoint(ep)
	}
	return writeValidatedRawConfig(path, replaceRawRange(pc.Raw, rangeInfo, replacement))
}

// DeleteEndpoint removes an endpoint by name.
func DeleteEndpoint(path, name string) error {
	pc, err := ParseConfigFile(path)
	if err != nil {
		return err
	}

	found := false
	for _, ep := range pc.Endpoints {
		if ep.Name == name {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("endpoint %q not found", name)
	}

	rangeInfo, err := findRawEndpointRange(pc.Raw, name)
	if err != nil {
		return err
	}
	return writeValidatedRawConfig(path, replaceRawRange(pc.Raw, rangeInfo, ""))
}

// ToggleEndpoint enables or disables an endpoint by name.
func ToggleEndpoint(path, name string, enabled bool) error {
	pc, err := ParseConfigFile(path)
	if err != nil {
		return err
	}

	found := false
	var target endpoints.Endpoint
	for i, ep := range pc.Endpoints {
		if ep.Name == name {
			if ep.Enabled == enabled {
				return nil
			}
			pc.Endpoints[i].Enabled = enabled
			target = pc.Endpoints[i]
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("endpoint %q not found", name)
	}

	rangeInfo, err := findRawEndpointRange(pc.Raw, name)
	if err != nil {
		return err
	}
	replacement := formatEndpoint(target)
	if !enabled {
		replacement = formatDisabledEndpoint(target)
	}
	return writeValidatedRawConfig(path, replaceRawRange(pc.Raw, rangeInfo, replacement))
}

type rawEndpointRange struct {
	start int
	end   int
}

func findRawEndpointRange(raw, name string) (rawEndpointRange, error) {
	lines := strings.SplitAfter(raw, "\n")
	offset := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == disabledBegin+name {
			start := offset
			end := offset + len(line)
			for _, candidate := range lines[i+1:] {
				end += len(candidate)
				if strings.TrimSpace(candidate) == disabledEnd+name {
					return rawEndpointRange{start: start, end: end}, nil
				}
			}
			return rawEndpointRange{}, fmt.Errorf("disabled endpoint %q has no end marker", name)
		}

		section := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
		parts := strings.SplitN(section, " ", 2)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") && len(parts) == 2 &&
			(parts[0] == "UdpEndpoint" || parts[0] == "UartEndpoint" || parts[0] == "TcpEndpoint") &&
			strings.TrimSpace(parts[1]) == name {
			start := offset
			end := offset + len(line)
			scanOffset := end
			for _, candidate := range lines[i+1:] {
				candidateTrimmed := strings.TrimSpace(candidate)
				if strings.HasPrefix(candidateTrimmed, "[") || strings.HasPrefix(candidateTrimmed, disabledBegin) {
					break
				}
				if candidateTrimmed != "" && !strings.HasPrefix(candidateTrimmed, "#") && !strings.HasPrefix(candidateTrimmed, ";") {
					end = scanOffset + len(candidate)
				}
				scanOffset += len(candidate)
			}
			return rawEndpointRange{start: start, end: end}, nil
		}
		offset += len(line)
	}
	return rawEndpointRange{}, fmt.Errorf("endpoint %q not found in raw config", name)
}

func replaceRawRange(raw string, target rawEndpointRange, replacement string) string {
	return raw[:target.start] + replacement + raw[target.end:]
}

func formatDisabledEndpoint(ep endpoints.Endpoint) string {
	var b strings.Builder
	b.WriteString(disabledBegin + ep.Name + "\n")
	for _, line := range strings.Split(strings.TrimSuffix(formatEndpoint(ep), "\n"), "\n") {
		if line == "" {
			b.WriteString("#\n")
		} else {
			b.WriteString("# " + line + "\n")
		}
	}
	b.WriteString(disabledEnd + ep.Name + "\n\n")
	return b.String()
}

func writeValidatedRawConfig(path, raw string) error {
	pc, err := ParseConfigText(raw, time.Now())
	if err != nil {
		return err
	}
	if err := ValidateParsedConfig(pc); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0644)
}
