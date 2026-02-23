package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

func (s *Server) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listEndpoints(w, r)
	case http.MethodPost:
		s.addEndpoint(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or POST only")
	}
}

func (s *Server) listEndpoints(w http.ResponseWriter, r *http.Request) {
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"endpoints": pc.Endpoints,
	})
}

func (s *Server) addEndpoint(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var ep endpoints.Endpoint
	if err := json.Unmarshal(body, &ep); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Validate
	if err := config.ValidateEndpointName(ep.Name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if ep.Type == "" {
		ep.Type = "UdpEndpoint"
	}
	if ep.Address != "" {
		if err := config.ValidateIP(ep.Address); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if ep.Port > 0 {
		if err := config.ValidatePort(ep.Port); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if ep.Mode == "" {
		ep.Mode = "normal"
	}
	if ep.Description == "" {
		ep.Description = endpoints.DescriptionForEndpoint(ep.Name, ep.Mode, ep.Address, ep.Port)
	}
	if ep.Category == "" {
		ep.Category = endpoints.CategoryForEndpoint(ep.Name, ep.Mode, ep.Address, ep.Port)
	}

	if err := config.AddEndpoint(s.configPath, ep); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "endpoint added", "name": ep.Name})
}

func (s *Server) handleEndpointByName(w http.ResponseWriter, r *http.Request) {
	name := extractEndpointName(r.URL.Path)
	if name == "" || name == "endpoints" {
		writeError(w, http.StatusBadRequest, "Endpoint name required")
		return
	}

	switch r.Method {
	case http.MethodPut:
		s.updateEndpoint(w, r, name)
	case http.MethodDelete:
		s.deleteEndpoint(w, r, name)
	case http.MethodPatch:
		s.toggleEndpoint(w, r, name)
	default:
		writeError(w, http.StatusMethodNotAllowed, "PUT, DELETE, or PATCH only")
	}
}

func (s *Server) updateEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var ep endpoints.Endpoint
	if err := json.Unmarshal(body, &ep); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	ep.Name = name

	if err := config.UpdateEndpoint(s.configPath, name, ep); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "endpoint updated"})
}

func (s *Server) deleteEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	if err := config.DeleteEndpoint(s.configPath, name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "endpoint deleted"})
}

func (s *Server) toggleEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var payload struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if err := config.ToggleEndpoint(s.configPath, name, payload.Enabled); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "endpoint toggled"})
}
