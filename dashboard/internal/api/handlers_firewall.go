package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/system"
)

func (s *Server) handleFirewall(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		info := system.GetFirewallInfo()
		writeJSON(w, http.StatusOK, info)
	case http.MethodPost:
		s.openFirewallPort(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

func (s *Server) openFirewallPort(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var payload struct {
		Port int `json:"port"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if payload.Port < 1 || payload.Port > 65535 {
		writeError(w, http.StatusBadRequest, "Invalid port number")
		return
	}

	if err := system.OpenFirewallPort(payload.Port); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to open port: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "port opened",
	})
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}
	info := system.DetectBoard()
	fw := system.GetFirewallInfo()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"board":    info,
		"firewall": fw,
	})
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"templates": endpoints_registry(),
	})
}
