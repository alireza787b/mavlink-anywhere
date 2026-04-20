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

	req, err := decodeProfileRequest(r.Body)
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

func decodeProfileRequest(body io.Reader) (profileRequest, error) {
	var req profileRequest
	raw, err := io.ReadAll(body)
	if err != nil {
		return profileRequest{}, err
	}
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
