package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/config"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/diagnostics"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/mavlink"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/system"
)

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	svc := system.GetServiceStatus()
	board := system.DetectBoard()
	firewall := system.GetFirewallInfo()
	pc, err := config.ParseConfigFile(s.configPath)

	snapshot := diagnostics.Collect(pc, svc, board, firewall, err)
	health := mavlink.Health{}
	if err == nil {
		health = mavlink.ProbeTCP(pc.General.TcpServerPort, 1500*time.Millisecond)
		if input, ok := findInputSource(pc); ok {
			health.Source = input
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mavlink": health,
		"alerts":  snapshot.Warnings,
		"docs":    snapshot.Docs,
		"firewall": firewall,
	})
}

func findInputSource(pc *config.ParsedConfig) (string, bool) {
	for _, ep := range pc.Endpoints {
		if ep.Type == "UartEndpoint" {
			return ep.Device + " @ " + strconvI(ep.Baud), true
		}
		if ep.Type == "UdpEndpoint" && strings.EqualFold(ep.Mode, "server") && ep.Name == "input" {
			return ep.Address + ":" + strconvI(ep.Port), true
		}
	}
	return "", false
}

func strconvI(v int) string {
	return fmt.Sprintf("%d", v)
}
