package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alireza787b/mavlink-anywhere/dashboard/web"
)

// Server holds shared state for all API handlers.
type Server struct {
	configPath string
	envPath    string
	version    string
}

// NewServer creates a new API server instance.
func NewServer(configPath, envPath, version string) *Server {
	return &Server{
		configPath: configPath,
		envPath:    envPath,
		version:    version,
	}
}

// Router returns the HTTP handler with all routes mounted.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// API v1 routes
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/diagnostics", s.handleDiagnostics)
	mux.HandleFunc("/api/v1/config", s.handleConfig)
	mux.HandleFunc("/api/v1/endpoints", s.handleEndpoints)
	mux.HandleFunc("/api/v1/endpoints/", s.handleEndpointByName)
	mux.HandleFunc("/api/v1/input", s.handleInput)
	mux.HandleFunc("/api/v1/profiles/export", s.handleProfilesExport)
	mux.HandleFunc("/api/v1/profiles/preview", s.handleProfilesPreview)
	mux.HandleFunc("/api/v1/profiles/apply", s.handleProfilesApply)
	mux.HandleFunc("/api/v1/profiles/backups", s.handleProfilesBackups)
	mux.HandleFunc("/api/v1/profiles/restore", s.handleProfilesRestore)
	mux.HandleFunc("/api/v1/service/restart", s.handleServiceRestart)
	mux.HandleFunc("/api/v1/service/stop", s.handleServiceStop)
	mux.HandleFunc("/api/v1/service/start", s.handleServiceStart)
	mux.HandleFunc("/api/v1/logs/stream", s.handleLogStream)
	mux.HandleFunc("/api/v1/logs/recent", s.handleLogsRecent)
	mux.HandleFunc("/api/v1/system/info", s.handleSystemInfo)
	mux.HandleFunc("/api/v1/system/firewall", s.handleFirewall)
	mux.HandleFunc("/api/v1/templates", s.handleTemplates)

	// Serve embedded static files
	staticFS := web.StaticFS()
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/", fileServer)

	return withCORS(mux)
}

// withCORS wraps a handler with CORS headers for development.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// extractEndpointName pulls the endpoint name from /api/v1/endpoints/{name}
func extractEndpointName(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
