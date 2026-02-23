package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

func (s *Server) handleInput(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getInput(w, r)
	case http.MethodPut:
		s.putInput(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "GET or PUT only")
	}
}

func (s *Server) getInput(w http.ResponseWriter, r *http.Request) {
	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
		return
	}

	// Find the input endpoint (UartEndpoint or server-mode UdpEndpoint named "input")
	var input *endpoints.Endpoint
	for _, ep := range pc.Endpoints {
		if ep.Type == "UartEndpoint" {
			epCopy := ep
			input = &epCopy
			break
		}
		if ep.Type == "UdpEndpoint" && ep.Name == "input" && ep.Mode == "server" {
			epCopy := ep
			input = &epCopy
			break
		}
	}

	env, _ := config.ParseEnvFile(s.envPath)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"input": input,
		"env":   env,
	})
}

func (s *Server) putInput(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body")
		return
	}

	var payload struct {
		Type    string `json:"type"`    // "uart" or "udp"
		Device  string `json:"device"`  // for uart
		Baud    int    `json:"baud"`    // for uart
		Address string `json:"address"` // for udp
		Port    int    `json:"port"`    // for udp
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	pc, err := config.ParseConfigFile(s.configPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
		return
	}

	// Remove existing input endpoints
	newEps := make([]endpoints.Endpoint, 0)
	for _, ep := range pc.Endpoints {
		if ep.Type == "UartEndpoint" {
			continue
		}
		if ep.Type == "UdpEndpoint" && ep.Name == "input" && ep.Mode == "server" {
			continue
		}
		newEps = append(newEps, ep)
	}

	// Add new input
	switch payload.Type {
	case "uart":
		if err := config.ValidateUartDevice(payload.Device); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if payload.Baud == 0 {
			payload.Baud = 57600
		}
		newEps = append([]endpoints.Endpoint{{
			Name:     "uart",
			Type:     "UartEndpoint",
			Device:   payload.Device,
			Baud:     payload.Baud,
			Category: "input",
			Enabled:  true,
		}}, newEps...)
	case "udp":
		if payload.Address == "" {
			payload.Address = "0.0.0.0"
		}
		if payload.Port == 0 {
			payload.Port = 14550
		}
		newEps = append([]endpoints.Endpoint{{
			Name:     "input",
			Type:     "UdpEndpoint",
			Mode:     "server",
			Address:  payload.Address,
			Port:     payload.Port,
			Category: "input",
			Enabled:  true,
		}}, newEps...)
	default:
		writeError(w, http.StatusBadRequest, "type must be 'uart' or 'udp'")
		return
	}

	pc.Endpoints = newEps
	if err := config.WriteConfigFile(s.configPath, pc); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to write config: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "input updated"})
}
