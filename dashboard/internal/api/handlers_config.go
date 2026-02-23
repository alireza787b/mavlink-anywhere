package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
)

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getConfig(w, r)
	case http.MethodPut:
		s.putConfig(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or PUT only")
	}
}

func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pc)
}

func (s *Server) putConfig(w http.ResponseWriter, r *http.Request) {
	// Raw config write (advanced mode)
	var payload struct {
		Raw string `json:"raw"`
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if payload.Raw == "" {
		writeError(w, http.StatusBadRequest, "Raw config cannot be empty")
		return
	}

	if err := os.WriteFile(s.configPath, []byte(payload.Raw), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "config updated"})
}
