package api

import (
	"net/http"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/system"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	svc := system.GetServiceStatus()
	board := system.DetectBoard()

	// Get config info
	var endpointCount int
	pc, err := config.ParseConfigFile(s.configPath)
	if err == nil {
		endpointCount = len(pc.Endpoints)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"version":       s.version,
		"service":       svc,
		"board":         board.BoardDesc,
		"architecture":  board.Arch,
		"hostname":      board.Hostname,
		"endpointCount": endpointCount,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
