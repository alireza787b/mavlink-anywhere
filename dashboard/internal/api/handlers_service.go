package api

import (
	"net/http"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/system"
)

func (s *Server) handleServiceRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if err := system.RestartService(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to restart service: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "service restarted"})
}

func (s *Server) handleServiceStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if err := system.StopService(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to stop service: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "service stopped"})
}

func (s *Server) handleServiceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "POST only")
		return
	}
	if err := system.StartService(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start service: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "service started"})
}
