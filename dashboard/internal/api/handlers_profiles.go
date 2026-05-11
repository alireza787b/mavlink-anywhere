package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/profiles"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/system"
)

type profileRequest struct {
	Mode    string           `json:"mode"`
	Profile profiles.Profile `json:"profile"`
}

type fleetProfileDiffRequest struct {
	Mode     string           `json:"mode"`
	Baseline profiles.Profile `json:"baseline"`
}

type fleetProfileValidateRequest struct {
	Baseline profiles.Profile `json:"baseline"`
	Profile  profiles.Profile `json:"profile"`
}

type fleetProfileApplyRequest struct {
	DryRunID     string `json:"dry_run_id"`
	Confirmation struct {
		Token             string `json:"token"`
		ConfirmationToken string `json:"confirmation_token"`
		AcknowledgedRisks bool   `json:"acknowledged_risks"`
		AdvancedStrictAck bool   `json:"advanced_strict_ack"`
		Operator          string `json:"operator"`
	} `json:"confirmation"`
}

func (s *Server) handleProfilesExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
		return
	}
	profile, err := profiles.Export(pc, s.version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to export profile: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleProfilesSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profiles.FleetSummary(pc, "node-local", s.configPath))
}

func (s *Server) handleProfilesValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req fleetProfileValidateRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	baseline := req.Baseline
	if len(baseline.Endpoints) == 0 && len(req.Profile.Endpoints) > 0 {
		baseline = req.Profile
	}
	if err := profiles.ValidateFleetBaseline(baseline); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"schema":  profiles.SidecarProfileSchema,
			"backend": "mavlink-anywhere",
			"valid":   false,
			"error":   err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schema":         profiles.SidecarProfileSchema,
		"backend":        "mavlink-anywhere",
		"valid":          true,
		"hash":           profiles.FleetBaselineHash(baseline),
		"hash_semantics": profiles.HashSemantics,
		"profile_count":  profiles.FleetBaselineEndpointCount(baseline),
	})
}

func (s *Server) handleProfilesDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req fleetProfileDiffRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse current config: "+err.Error())
		return
	}
	diff, err := profiles.FleetDiff(pc, req.Baseline, req.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, diff)
}

func (s *Server) handleProfilesImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	var req profiles.FleetProfileRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !req.DryRun {
		writeError(w, http.StatusBadRequest, "profile import requires dry_run=true; use apply with confirmation")
		return
	}
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse current config: "+err.Error())
		return
	}
	plan, err := profiles.DryRunFleetPlan(pc, req.Baseline, req.Mode, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dryRunID, _ := plan["dry_run_id"].(string)
	s.mu.Lock()
	s.plans[dryRunID] = plan
	s.mu.Unlock()
	publicPlan := map[string]any{}
	for key, value := range plan {
		if key == "candidate_config" {
			continue
		}
		publicPlan[key] = value
	}
	writeJSON(w, http.StatusOK, publicPlan)
}

func (s *Server) handleProfilesPromoteReferenceDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
		return
	}
	draft, err := profiles.FleetReferenceDraft(pc, s.version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to promote reference draft: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (s *Server) handleProfilesPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	req, err := decodeProfileRequest(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse current config: "+err.Error())
		return
	}
	preview, err := profiles.PreviewApply(pc, req.Profile, req.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleProfilesApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var fleetReq fleetProfileApplyRequest
	if err := json.Unmarshal(raw, &fleetReq); err == nil && fleetReq.DryRunID != "" {
		s.handleFleetProfilesApply(w, fleetReq)
		return
	}
	req, err := decodeProfileRequestBytes(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	current, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse current config: "+err.Error())
		return
	}

	preview, err := profiles.PreviewApply(current, req.Profile, req.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	backup, _, err := profiles.Apply(s.configPath, s.envPath, current, req.Profile, req.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := system.RestartService(); err != nil {
		rollbackErr := rollbackLatestBackup(s.configPath, s.envPath)
		if rollbackErr != nil {
			writeError(w, http.StatusInternalServerError, "Profile applied but service restart failed, and rollback failed: "+rollbackErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "Service restart failed after applying profile; previous config was restored")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "profile applied",
		"preview": preview,
		"backup":  backup,
	})
}

func (s *Server) handleFleetProfilesApply(w http.ResponseWriter, req fleetProfileApplyRequest) {
	if req.DryRunID == "" || !req.Confirmation.AcknowledgedRisks {
		writeError(w, http.StatusBadRequest, "dry_run_id and acknowledged_risks are required")
		return
	}
	s.mu.Lock()
	plan := s.plans[req.DryRunID]
	s.mu.Unlock()
	if plan == nil {
		writeError(w, http.StatusNotFound, "dry-run plan not found")
		return
	}
	confirmToken := req.Confirmation.Token
	if confirmToken == "" {
		confirmToken = req.Confirmation.ConfirmationToken
	}
	if plan["confirmation_token"] != confirmToken {
		writeError(w, http.StatusBadRequest, "confirmation token does not match dry-run plan")
		return
	}
	if plan["requires_advanced_confirmation"] == true && !req.Confirmation.AdvancedStrictAck {
		writeError(w, http.StatusBadRequest, "fleet-strict requires advanced confirmation")
		return
	}
	backup, target, err := profiles.ApplyFleetPlan(s.configPath, s.envPath, plan, confirmToken)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := system.RestartService(); err != nil {
		rollbackErr := rollbackLatestBackup(s.configPath, s.envPath)
		if rollbackErr != nil {
			writeError(w, http.StatusInternalServerError, "Profile applied but service restart failed, and rollback failed: "+rollbackErr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "Service restart failed after applying profile; previous config was restored")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schema":         profiles.SidecarProfileSchema,
		"backend":        "mavlink-anywhere",
		"applied":        true,
		"mode":           plan["mode"],
		"dry_run_id":     req.DryRunID,
		"candidate_hash": profiles.FleetHash(target),
		"backup":         backup,
	})
}

func (s *Server) handleProfilesBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	backups, err := profiles.ListBackups(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list backups: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"backups": backups,
	})
}

func (s *Server) handleProfilesRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}

	backup, err := profiles.RestoreLatest(s.configPath, s.envPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := system.RestartService(); err != nil {
		writeError(w, http.StatusInternalServerError, "Backup restored but service restart failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "backup restored",
		"backup": backup,
	})
}

func decodeJSON(body io.Reader, target any) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func decodeProfileRequest(body io.Reader) (profileRequest, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return profileRequest{}, err
	}
	return decodeProfileRequestBytes(raw)
}

func decodeProfileRequestBytes(raw []byte) (profileRequest, error) {
	var req profileRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return profileRequest{}, err
	}
	return req, nil
}

func rollbackLatestBackup(configPath, envPath string) error {
	if _, err := profiles.RestoreLatest(configPath, envPath); err != nil {
		return err
	}
	return system.RestartService()
}
