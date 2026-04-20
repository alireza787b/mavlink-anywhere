package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

const (
	SchemaVersion = "1"
	Kind          = "mavlink-anywhere-profile"

	ModeReplace = "replace"
	ModeMerge   = "merge_endpoints"
)

type Metadata struct {
	ProfileName string `json:"profileName"`
	Description string `json:"description,omitempty"`
	ExportedAt  string `json:"exportedAt"`
	ExportedBy  string `json:"exportedBy"`
	Hostname    string `json:"hostname,omitempty"`
}

type Profile struct {
	SchemaVersion string                `json:"schemaVersion"`
	Kind          string                `json:"kind"`
	Metadata      Metadata              `json:"metadata"`
	General       config.GeneralSection `json:"general"`
	Endpoints     []endpoints.Endpoint  `json:"endpoints"`
}

type ChangeSet struct {
	InputChanged   bool     `json:"inputChanged"`
	GeneralChanged bool     `json:"generalChanged"`
	Added          []string `json:"added"`
	Updated        []string `json:"updated"`
	Removed        []string `json:"removed"`
	Preserved      []string `json:"preserved"`
}

type Preview struct {
	Mode            string    `json:"mode"`
	RestartRequired bool      `json:"restartRequired"`
	RebootRequired  bool      `json:"rebootRequired"`
	Summary         []string  `json:"summary"`
	Warnings        []string  `json:"warnings"`
	Changes         ChangeSet `json:"changes"`
	Profile         Profile   `json:"profile"`
}

type BackupInfo struct {
	ID           string `json:"id"`
	CreatedAt    string `json:"createdAt"`
	ConfigPath   string `json:"configPath"`
	EnvPath      string `json:"envPath"`
	ConfigBackup string `json:"configBackup"`
	EnvBackup    string `json:"envBackup"`
	MetadataPath string `json:"metadataPath"`
}

type backupMetadata struct {
	BackupInfo
}

func Export(pc *config.ParsedConfig, version string) (Profile, error) {
	if pc == nil {
		return Profile{}, fmt.Errorf("parsed config is required")
	}
	hostname, _ := os.Hostname()
	return Profile{
		SchemaVersion: SchemaVersion,
		Kind:          Kind,
		Metadata: Metadata{
			ProfileName: defaultProfileName(hostname),
			ExportedAt:  time.Now().UTC().Format(time.RFC3339),
			ExportedBy:  version,
			Hostname:    hostname,
		},
		General:   pc.General,
		Endpoints: cloneEndpoints(pc.Endpoints),
	}, nil
}

func ParseJSON(raw []byte) (Profile, error) {
	var profile Profile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func Validate(profile Profile) error {
	if strings.TrimSpace(profile.SchemaVersion) != SchemaVersion {
		return fmt.Errorf("unsupported profile schema version: %q", profile.SchemaVersion)
	}
	if strings.TrimSpace(profile.Kind) != Kind {
		return fmt.Errorf("unsupported profile kind: %q", profile.Kind)
	}
	if len(profile.Endpoints) == 0 {
		return fmt.Errorf("profile must include at least one endpoint")
	}
	if profile.General.TcpServerPort < 0 || profile.General.TcpServerPort > 65535 {
		return fmt.Errorf("invalid TcpServerPort: %d", profile.General.TcpServerPort)
	}

	seen := map[string]struct{}{}
	inputCount := 0
	validated := make([]endpoints.Endpoint, 0, len(profile.Endpoints))
	for _, ep := range profile.Endpoints {
		if err := validateEndpoint(ep); err != nil {
			return err
		}
		if _, ok := seen[ep.Name]; ok {
			return fmt.Errorf("duplicate endpoint name in profile: %q", ep.Name)
		}
		seen[ep.Name] = struct{}{}
		if isInputEndpoint(ep) {
			inputCount++
		}
		if err := config.ValidateEndpointTopology(validated, ep, ""); err != nil {
			return err
		}
		validated = append(validated, ep)
	}
	if inputCount == 0 {
		return fmt.Errorf("profile must include exactly one input endpoint")
	}
	if inputCount > 1 {
		return fmt.Errorf("profile contains multiple input endpoints")
	}
	return nil
}

func PreviewApply(current *config.ParsedConfig, profile Profile, mode string) (Preview, error) {
	if current == nil {
		return Preview{}, fmt.Errorf("current config is required")
	}
	if err := Validate(profile); err != nil {
		return Preview{}, err
	}
	target, err := buildTargetConfig(current, profile, mode)
	if err != nil {
		return Preview{}, err
	}

	changes := compareConfigs(current, target)
	summary := buildSummary(mode, changes)
	warnings := buildWarnings(current, target, mode, changes)

	return Preview{
		Mode:            normalizeMode(mode),
		RestartRequired: true,
		RebootRequired:  false,
		Summary:         summary,
		Warnings:        warnings,
		Changes:         changes,
		Profile:         profile,
	}, nil
}

func Apply(configPath, envPath string, current *config.ParsedConfig, profile Profile, mode string) (BackupInfo, *config.ParsedConfig, error) {
	if current == nil {
		return BackupInfo{}, nil, fmt.Errorf("current config is required")
	}
	if err := Validate(profile); err != nil {
		return BackupInfo{}, nil, err
	}
	target, err := buildTargetConfig(current, profile, mode)
	if err != nil {
		return BackupInfo{}, nil, err
	}

	backup, err := CreateBackup(configPath, envPath)
	if err != nil {
		return BackupInfo{}, nil, err
	}
	if err := config.WriteConfigAndEnv(configPath, envPath, target); err != nil {
		return BackupInfo{}, nil, err
	}
	return backup, target, nil
}

func CreateBackup(configPath, envPath string) (BackupInfo, error) {
	backupID := time.Now().UTC().Format("20060102T150405Z")
	backupDir := filepath.Join(filepath.Dir(configPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return BackupInfo{}, err
	}

	info := BackupInfo{
		ID:           backupID,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		ConfigPath:   configPath,
		EnvPath:      envPath,
		ConfigBackup: filepath.Join(backupDir, fmt.Sprintf("main-%s.conf", backupID)),
		EnvBackup:    filepath.Join(backupDir, fmt.Sprintf("mavlink-router-%s.env", backupID)),
		MetadataPath: filepath.Join(backupDir, fmt.Sprintf("backup-%s.json", backupID)),
	}

	if err := copyFile(configPath, info.ConfigBackup); err != nil {
		return BackupInfo{}, err
	}
	if _, err := os.Stat(envPath); err == nil {
		if err := copyFile(envPath, info.EnvBackup); err != nil {
			return BackupInfo{}, err
		}
	}
	metaBytes, err := json.MarshalIndent(backupMetadata{BackupInfo: info}, "", "  ")
	if err != nil {
		return BackupInfo{}, err
	}
	if err := os.WriteFile(info.MetadataPath, metaBytes, 0644); err != nil {
		return BackupInfo{}, err
	}
	return info, nil
}

func ListBackups(configPath string) ([]BackupInfo, error) {
	backupDir := filepath.Join(filepath.Dir(configPath), "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupInfo{}, nil
		}
		return nil, err
	}

	backups := make([]BackupInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "backup-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(backupDir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var meta backupMetadata
		if err := json.Unmarshal(raw, &meta); err != nil {
			return nil, err
		}
		backups = append(backups, meta.BackupInfo)
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt > backups[j].CreatedAt
	})
	return backups, nil
}

func RestoreLatest(configPath, envPath string) (BackupInfo, error) {
	backups, err := ListBackups(configPath)
	if err != nil {
		return BackupInfo{}, err
	}
	if len(backups) == 0 {
		return BackupInfo{}, fmt.Errorf("no backups available")
	}
	latest := backups[0]
	if err := copyFile(latest.ConfigBackup, configPath); err != nil {
		return BackupInfo{}, err
	}
	if _, err := os.Stat(latest.EnvBackup); err == nil {
		if err := copyFile(latest.EnvBackup, envPath); err != nil {
			return BackupInfo{}, err
		}
	} else if err := config.SyncEnvFromConfig(configPath, envPath); err != nil {
		return BackupInfo{}, err
	}
	return latest, nil
}

func buildTargetConfig(current *config.ParsedConfig, profile Profile, mode string) (*config.ParsedConfig, error) {
	normalizedMode := normalizeMode(mode)
	switch normalizedMode {
	case ModeReplace:
		return &config.ParsedConfig{
			General:   profile.General,
			Endpoints: cloneEndpoints(profile.Endpoints),
		}, nil
	case ModeMerge:
		return mergeConfig(current, profile)
	default:
		return nil, fmt.Errorf("unsupported profile mode: %q", mode)
	}
}

func mergeConfig(current *config.ParsedConfig, profile Profile) (*config.ParsedConfig, error) {
	target := &config.ParsedConfig{
		General:   current.General,
		Endpoints: []endpoints.Endpoint{},
	}

	incomingInputs := make([]endpoints.Endpoint, 0, 1)
	incomingByName := make(map[string]endpoints.Endpoint)
	for _, ep := range profile.Endpoints {
		if isInputEndpoint(ep) {
			incomingInputs = append(incomingInputs, ep)
			continue
		}
		incomingByName[ep.Name] = ep
	}

	if len(incomingInputs) == 0 {
		return nil, fmt.Errorf("merge profile must include an input endpoint")
	}

	target.Endpoints = append(target.Endpoints, cloneEndpoints(incomingInputs)...)

	used := map[string]struct{}{}
	for _, currentEP := range current.Endpoints {
		if isInputEndpoint(currentEP) {
			continue
		}
		if incoming, ok := incomingByName[currentEP.Name]; ok {
			target.Endpoints = append(target.Endpoints, incoming)
			used[currentEP.Name] = struct{}{}
			continue
		}
		target.Endpoints = append(target.Endpoints, currentEP)
	}
	for name, incoming := range incomingByName {
		if _, ok := used[name]; ok {
			continue
		}
		target.Endpoints = append(target.Endpoints, incoming)
	}

	if err := Validate(Profile{
		SchemaVersion: SchemaVersion,
		Kind:          Kind,
		Metadata:      profile.Metadata,
		General:       target.General,
		Endpoints:     target.Endpoints,
	}); err != nil {
		return nil, err
	}
	return target, nil
}

func compareConfigs(current, target *config.ParsedConfig) ChangeSet {
	changes := ChangeSet{
		Added:     []string{},
		Updated:   []string{},
		Removed:   []string{},
		Preserved: []string{},
	}

	if current.General != target.General {
		changes.GeneralChanged = true
	}
	currentInput, _ := findInput(current.Endpoints)
	targetInput, _ := findInput(target.Endpoints)
	if !endpointsEqual(currentInput, targetInput) {
		changes.InputChanged = true
	}

	currentByName := map[string]endpoints.Endpoint{}
	targetByName := map[string]endpoints.Endpoint{}
	for _, ep := range current.Endpoints {
		if isInputEndpoint(ep) {
			continue
		}
		currentByName[ep.Name] = ep
	}
	for _, ep := range target.Endpoints {
		if isInputEndpoint(ep) {
			continue
		}
		targetByName[ep.Name] = ep
	}

	for name, currentEP := range currentByName {
		targetEP, ok := targetByName[name]
		if !ok {
			changes.Removed = append(changes.Removed, name)
			continue
		}
		if endpointsEqual(currentEP, targetEP) {
			changes.Preserved = append(changes.Preserved, name)
		} else {
			changes.Updated = append(changes.Updated, name)
		}
	}
	for name := range targetByName {
		if _, ok := currentByName[name]; !ok {
			changes.Added = append(changes.Added, name)
		}
	}

	sort.Strings(changes.Added)
	sort.Strings(changes.Updated)
	sort.Strings(changes.Removed)
	sort.Strings(changes.Preserved)
	return changes
}

func buildSummary(mode string, changes ChangeSet) []string {
	summary := []string{}
	if changes.InputChanged {
		summary = append(summary, "Input source will change.")
	}
	if changes.GeneralChanged {
		summary = append(summary, "General router settings will change.")
	}
	if len(changes.Added) > 0 {
		summary = append(summary, fmt.Sprintf("%d endpoint(s) will be added.", len(changes.Added)))
	}
	if len(changes.Updated) > 0 {
		summary = append(summary, fmt.Sprintf("%d endpoint(s) will be updated.", len(changes.Updated)))
	}
	if len(changes.Removed) > 0 {
		summary = append(summary, fmt.Sprintf("%d endpoint(s) will be removed.", len(changes.Removed)))
	}
	if normalizeMode(mode) == ModeMerge {
		summary = append(summary, "Merge mode preserves existing endpoints that are not named in the imported profile.")
	}
	if len(summary) == 0 {
		summary = append(summary, "Imported profile matches the current effective routing profile.")
	}
	return summary
}

func buildWarnings(current, target *config.ParsedConfig, mode string, changes ChangeSet) []string {
	warnings := []string{}
	if normalizeMode(mode) == ModeReplace && len(changes.Removed) > 0 {
		warnings = append(warnings, "Replace mode removes endpoints not present in the imported profile. A backup will be created automatically.")
	}
	if changes.InputChanged {
		warnings = append(warnings, "Applying the profile will restart mavlink-router. Reboot is not required unless you separately change host serial boot settings.")
	}
	hasGCSListen := false
	hasPush14550 := false
	for _, ep := range target.Endpoints {
		if ep.Type != "UdpEndpoint" || !ep.Enabled {
			continue
		}
		if strings.EqualFold(ep.Mode, "server") && ep.Port == 14550 {
			hasGCSListen = true
		}
		if strings.EqualFold(ep.Mode, "normal") && ep.Port == 14550 && !isLoopback(ep.Address) {
			hasPush14550 = true
		}
	}
	if hasGCSListen && hasPush14550 {
		warnings = append(warnings, "Listener mode on 14550 and explicit push to a remote 14550 can coexist, but the same remote GCS should not consume both simultaneously.")
	}
	return warnings
}

func validateEndpoint(ep endpoints.Endpoint) error {
	if err := config.ValidateEndpointName(ep.Name); err != nil {
		return err
	}
	switch ep.Type {
	case "UartEndpoint":
		if err := config.ValidateUartDevice(ep.Device); err != nil {
			return err
		}
		if err := config.ValidateBaud(ep.Baud); err != nil {
			return err
		}
	case "UdpEndpoint":
		if err := config.ValidateEndpointMode(ep.Mode); err != nil {
			return err
		}
		if err := config.ValidateIP(ep.Address); err != nil {
			return err
		}
		if err := config.ValidatePort(ep.Port); err != nil {
			return err
		}
	case "TcpEndpoint":
		if err := config.ValidateIP(ep.Address); err != nil {
			return err
		}
		if err := config.ValidatePort(ep.Port); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported endpoint type: %q", ep.Type)
	}
	return nil
}

func normalizeMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", ModeReplace:
		return ModeReplace
	case ModeMerge:
		return ModeMerge
	default:
		return mode
	}
}

func cloneEndpoints(src []endpoints.Endpoint) []endpoints.Endpoint {
	cloned := make([]endpoints.Endpoint, len(src))
	copy(cloned, src)
	return cloned
}

func defaultProfileName(hostname string) string {
	base := "routing-profile"
	if strings.TrimSpace(hostname) != "" {
		base = hostname + "-routing-profile"
	}
	return base
}

func isInputEndpoint(ep endpoints.Endpoint) bool {
	if ep.Type == "UartEndpoint" {
		return true
	}
	return ep.Type == "UdpEndpoint" && strings.EqualFold(ep.Mode, "server") && ep.Name == "input"
}

func findInput(items []endpoints.Endpoint) (endpoints.Endpoint, bool) {
	for _, ep := range items {
		if isInputEndpoint(ep) {
			return ep, true
		}
	}
	return endpoints.Endpoint{}, false
}

func endpointsEqual(a, b endpoints.Endpoint) bool {
	return a.Name == b.Name &&
		a.Type == b.Type &&
		strings.EqualFold(a.Mode, b.Mode) &&
		a.Address == b.Address &&
		a.Port == b.Port &&
		a.Device == b.Device &&
		a.Baud == b.Baud &&
		a.Enabled == b.Enabled
}

func copyFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0644)
}

func isLoopback(addr string) bool {
	return addr == "127.0.0.1" || addr == "localhost"
}
