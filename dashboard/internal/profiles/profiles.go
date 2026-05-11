package profiles

import (
	"crypto/sha256"
	"encoding/hex"
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

	SidecarProfileSchema = "mds.sidecar_profile.v1"
	HashSemantics        = "sha256:canonical-sanitized-payload:12"
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

type FleetProfileRequest struct {
	Mode     string  `json:"mode"`
	DryRun   bool    `json:"dry_run"`
	Baseline Profile `json:"baseline"`
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

func NormalizePolicyMode(mode string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		normalized = "local"
	}
	switch normalized {
	case "observe", "local", "fleet-merge", "fleet-strict":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported profile mode: %q", mode)
	}
}

func FleetSummary(pc *config.ParsedConfig, source string, path string) map[string]any {
	if pc == nil {
		return map[string]any{
			"schema":         SidecarProfileSchema,
			"backend":        "mavlink-anywhere",
			"kind":           Kind,
			"source":         source,
			"path":           path,
			"present":        false,
			"hash":           nil,
			"hash_semantics": HashSemantics,
			"profile_count":  0,
			"secret_status":  "missing",
			"endpoints":      []map[string]any{},
			"overlay":        map[string]any{},
		}
	}
	return map[string]any{
		"schema":         SidecarProfileSchema,
		"backend":        "mavlink-anywhere",
		"kind":           Kind,
		"source":         source,
		"path":           path,
		"present":        true,
		"hash":           FleetHash(pc),
		"hash_semantics": HashSemantics,
		"profile_count":  len(nonInputEndpoints(pc.Endpoints)),
		"secret_status":  "missing",
		"endpoints":      sanitizedEndpoints(nonInputEndpoints(pc.Endpoints)),
		"overlay": map[string]any{
			"hardware_source": sanitizedEndpoints(inputEndpoints(pc.Endpoints)),
			"overlay_hash":    overlayHash(pc),
		},
	}
}

func FleetReferenceDraft(pc *config.ParsedConfig, version string) (map[string]any, error) {
	profile, err := Export(pc, version)
	if err != nil {
		return nil, err
	}
	profile.Endpoints = nonInputEndpoints(profile.Endpoints)
	return map[string]any{
		"schema":     SidecarProfileSchema,
		"backend":    "mavlink-anywhere",
		"kind":       Kind,
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"profile":    profile,
		"summary":    FleetSummary(&config.ParsedConfig{General: profile.General, Endpoints: profile.Endpoints}, "reference-draft", ""),
	}, nil
}

func FleetHash(pc *config.ParsedConfig) string {
	data, _ := json.Marshal(canonicalFleetPayload(pc))
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

func FleetDiff(current *config.ParsedConfig, baseline Profile, mode string) (map[string]any, error) {
	normalizedMode, err := NormalizePolicyMode(mode)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("current config is required")
	}
	if err := validateFleetBaseline(baseline); err != nil {
		return nil, err
	}
	target, err := MergeFleetPolicy(current, baseline, normalizedMode)
	if err != nil {
		return nil, err
	}
	changes := comparePolicyEndpoints(current, target)
	baselinePolicy := nonInputEndpoints(baseline.Endpoints)
	driftState := "outdated"
	switch {
	case normalizedMode == "observe" || normalizedMode == "local":
		driftState = "unmanaged"
	case len(baselinePolicy) == 0 && len(nonInputEndpoints(current.Endpoints)) > 0:
		driftState = "missing_fleet_baseline"
	case len(baselinePolicy) == 0:
		driftState = "unmanaged"
	case len(changes.Added) == 0 && len(changes.Updated) == 0 && len(changes.Removed) == 0:
		driftState = "in_sync"
	case normalizedMode == "fleet-merge" && len(changes.Removed) > 0 && len(changes.Added) == 0 && len(changes.Updated) == 0:
		driftState = "local_extra"
	}
	strictPrune := []string{}
	preserveLocal := []string{}
	if normalizedMode == "fleet-strict" {
		strictPrune = append(strictPrune, changes.Removed...)
	}
	if normalizedMode == "fleet-merge" {
		preserveLocal = append(preserveLocal, changes.Removed...)
	}
	return map[string]any{
		"schema":        SidecarProfileSchema,
		"backend":       "mavlink-anywhere",
		"mode":          normalizedMode,
		"drift_state":   driftState,
		"local_hash":    FleetHash(current),
		"baseline_hash": profilePolicyHash(baseline),
		"overlay_hash":  overlayHash(current),
		"changes": map[string]any{
			"add_from_baseline":    changes.Added,
			"update_from_baseline": changes.Updated,
			"local_extra":          changes.Removed,
			"strict_prune":         strictPrune,
			"preserve_local":       preserveLocal,
		},
		"warnings": fleetWarnings(normalizedMode, changes),
	}, nil
}

func MergeFleetPolicy(current *config.ParsedConfig, baseline Profile, mode string) (*config.ParsedConfig, error) {
	normalizedMode, err := NormalizePolicyMode(mode)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("current config is required")
	}
	if normalizedMode == "observe" || normalizedMode == "local" {
		return cloneParsedConfig(current), nil
	}
	if err := validateFleetBaseline(baseline); err != nil {
		return nil, err
	}
	target := &config.ParsedConfig{
		General:   baseline.General,
		Endpoints: []endpoints.Endpoint{},
	}
	target.Endpoints = append(target.Endpoints, cloneEndpoints(inputEndpoints(current.Endpoints))...)
	baselineByName := map[string]endpoints.Endpoint{}
	for _, ep := range nonInputEndpoints(baseline.Endpoints) {
		baselineByName[ep.Name] = ep
		target.Endpoints = append(target.Endpoints, ep)
	}
	if normalizedMode == "fleet-merge" {
		for _, ep := range nonInputEndpoints(current.Endpoints) {
			if _, exists := baselineByName[ep.Name]; !exists {
				target.Endpoints = append(target.Endpoints, ep)
			}
		}
	}
	return target, nil
}

func DryRunFleetPlan(current *config.ParsedConfig, baseline Profile, mode string, includeCandidate bool) (map[string]any, error) {
	normalizedMode, err := NormalizePolicyMode(mode)
	if err != nil {
		return nil, err
	}
	if err := validateFleetBaseline(baseline); err != nil {
		return nil, err
	}
	candidate, err := MergeFleetPolicy(current, baseline, normalizedMode)
	if err != nil {
		return nil, err
	}
	diff, err := FleetDiff(current, baseline, normalizedMode)
	if err != nil {
		return nil, err
	}
	seed := map[string]any{
		"mode":      normalizedMode,
		"local":     FleetHash(current),
		"baseline":  profilePolicyHash(baseline),
		"candidate": FleetHash(candidate),
		"created":   time.Now().Unix(),
	}
	rawSeed, _ := json.Marshal(seed)
	sum := sha256.Sum256(rawSeed)
	token := hex.EncodeToString(sum[:])[:16]
	plan := map[string]any{
		"schema":                         SidecarProfileSchema,
		"backend":                        "mavlink-anywhere",
		"kind":                           "mavlink-anywhere-profile-plan",
		"dry_run_id":                     "mla-" + token[:12],
		"created_at":                     time.Now().UTC().Format(time.RFC3339),
		"mode":                           normalizedMode,
		"confirmation_token":             token,
		"requires_confirmation":          normalizedMode != "observe" && normalizedMode != "local",
		"requires_advanced_confirmation": normalizedMode == "fleet-strict",
		"diff":                           diff,
		"candidate_hash":                 FleetHash(candidate),
		"candidate_summary":              FleetSummary(candidate, "candidate", ""),
	}
	if includeCandidate {
		plan["candidate_config"] = candidate
	}
	return plan, nil
}

func ApplyFleetPlan(configPath, envPath string, plan map[string]any, confirm string) (BackupInfo, *config.ParsedConfig, error) {
	expected, _ := plan["confirmation_token"].(string)
	if expected == "" || confirm != expected {
		return BackupInfo{}, nil, fmt.Errorf("confirmation token does not match dry-run plan")
	}
	mode, _ := plan["mode"].(string)
	if mode == "observe" || mode == "local" {
		return BackupInfo{}, nil, fmt.Errorf("%s mode does not produce apply mutations", mode)
	}
	candidate, ok := plan["candidate_config"].(*config.ParsedConfig)
	if !ok {
		return BackupInfo{}, nil, fmt.Errorf("dry-run plan missing candidate config")
	}
	backup, err := CreateBackup(configPath, envPath)
	if err != nil {
		return BackupInfo{}, nil, err
	}
	if err := config.WriteConfigAndEnv(configPath, envPath, candidate); err != nil {
		return BackupInfo{}, nil, err
	}
	return backup, candidate, nil
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

func cloneParsedConfig(src *config.ParsedConfig) *config.ParsedConfig {
	if src == nil {
		return nil
	}
	return &config.ParsedConfig{
		General:    src.General,
		Endpoints:  cloneEndpoints(src.Endpoints),
		Raw:        src.Raw,
		ModifiedAt: src.ModifiedAt,
	}
}

func inputEndpoints(items []endpoints.Endpoint) []endpoints.Endpoint {
	result := []endpoints.Endpoint{}
	for _, ep := range items {
		if isInputEndpoint(ep) {
			result = append(result, ep)
		}
	}
	return result
}

func nonInputEndpoints(items []endpoints.Endpoint) []endpoints.Endpoint {
	result := []endpoints.Endpoint{}
	for _, ep := range items {
		if !isInputEndpoint(ep) {
			result = append(result, ep)
		}
	}
	return result
}

func sanitizedEndpoints(items []endpoints.Endpoint) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, ep := range items {
		result = append(result, map[string]any{
			"name":     ep.Name,
			"type":     ep.Type,
			"mode":     strings.ToLower(ep.Mode),
			"address":  ep.Address,
			"port":     ep.Port,
			"device":   ep.Device,
			"baud":     ep.Baud,
			"category": ep.Category,
			"enabled":  ep.Enabled,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		left, _ := result[i]["name"].(string)
		right, _ := result[j]["name"].(string)
		return strings.ToLower(left) < strings.ToLower(right)
	})
	return result
}

func canonicalFleetPayload(pc *config.ParsedConfig) map[string]any {
	if pc == nil {
		return map[string]any{"general": config.GeneralSection{}, "endpoints": []map[string]any{}}
	}
	return map[string]any{
		"general":   pc.General,
		"endpoints": sanitizedEndpoints(nonInputEndpoints(pc.Endpoints)),
	}
}

func overlayHash(pc *config.ParsedConfig) string {
	payload := map[string]any{"hardware_source": sanitizedEndpoints(inputEndpoints(pc.Endpoints))}
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

func profilePolicyHash(profile Profile) string {
	pc := &config.ParsedConfig{General: profile.General, Endpoints: nonInputEndpoints(profile.Endpoints)}
	return FleetHash(pc)
}

func FleetBaselineHash(profile Profile) string {
	return profilePolicyHash(profile)
}

func FleetBaselineEndpointCount(profile Profile) int {
	return len(nonInputEndpoints(profile.Endpoints))
}

func ValidateFleetBaseline(profile Profile) error {
	return validateFleetBaseline(profile)
}

func validateFleetBaseline(profile Profile) error {
	if strings.TrimSpace(profile.Kind) != "" && strings.TrimSpace(profile.Kind) != Kind {
		return fmt.Errorf("unsupported profile kind: %q", profile.Kind)
	}
	seen := map[string]struct{}{}
	validated := []endpoints.Endpoint{}
	for _, ep := range nonInputEndpoints(profile.Endpoints) {
		if err := validateEndpoint(ep); err != nil {
			return err
		}
		if _, ok := seen[ep.Name]; ok {
			return fmt.Errorf("duplicate endpoint name in profile: %q", ep.Name)
		}
		seen[ep.Name] = struct{}{}
		if err := config.ValidateEndpointTopology(validated, ep, ""); err != nil {
			return err
		}
		validated = append(validated, ep)
	}
	return nil
}

func comparePolicyEndpoints(current, target *config.ParsedConfig) ChangeSet {
	changes := ChangeSet{
		Added:     []string{},
		Updated:   []string{},
		Removed:   []string{},
		Preserved: []string{},
	}
	if current == nil || target == nil {
		return changes
	}
	if current.General != target.General {
		changes.GeneralChanged = true
	}
	currentByName := map[string]endpoints.Endpoint{}
	targetByName := map[string]endpoints.Endpoint{}
	for _, ep := range nonInputEndpoints(current.Endpoints) {
		currentByName[ep.Name] = ep
	}
	for _, ep := range nonInputEndpoints(target.Endpoints) {
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

func fleetWarnings(mode string, changes ChangeSet) []string {
	warnings := []string{}
	if mode == "observe" {
		warnings = append(warnings, "observe mode reports only and will not apply routing changes")
	}
	if mode == "local" {
		warnings = append(warnings, "local mode keeps the node-local routing profile authoritative")
	}
	if mode == "fleet-merge" && len(changes.Removed) > 0 {
		warnings = append(warnings, "fleet-merge preserves node-local endpoints not present in the fleet baseline")
	}
	if mode == "fleet-strict" && len(changes.Removed) > 0 {
		warnings = append(warnings, "fleet-strict removes non-baseline output endpoints but preserves the hardware input overlay")
	}
	return warnings
}
