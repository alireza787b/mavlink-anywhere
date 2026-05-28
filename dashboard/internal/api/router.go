package api

import (
	"crypto/subtle"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/alireza787b/mavlink-anywhere/dashboard/web"
	"golang.org/x/crypto/bcrypt"
)

// Server holds shared state for all API handlers.
type Server struct {
	configPath string
	envPath    string
	version    string
	mu         sync.Mutex
	plans      map[string]map[string]any
}

const (
	mutationTokenEnv        = "MAVLINK_ANYWHERE_API_TOKEN"
	dashboardAuthUserEnv    = "MAVLINK_ANYWHERE_DASHBOARD_USER"
	dashboardAuthBcryptEnv  = "MAVLINK_ANYWHERE_DASHBOARD_PASSWORD_BCRYPT"
	openMutationsEnv        = "MAVLINK_ANYWHERE_ALLOW_UNAUTHENTICATED_MUTATIONS"
	dashboardCSRFHeader     = "X-Sidecar-CSRF"
	corsAllowedOriginEnv    = "MAVLINK_ANYWHERE_CORS_ALLOWED_ORIGIN"
	dashboardAuthBasicRealm = `Basic realm="MAVLink Anywhere", charset="UTF-8"`
)

// NewServer creates a new API server instance.
func NewServer(configPath, envPath, version string) *Server {
	return &Server{
		configPath: configPath,
		envPath:    envPath,
		version:    version,
		plans:      map[string]map[string]any{},
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
	mux.HandleFunc("/api/v1/profiles/summary", s.handleProfilesSummary)
	mux.HandleFunc("/api/v1/profiles/validate", s.handleProfilesValidate)
	mux.HandleFunc("/api/v1/profiles/diff", s.handleProfilesDiff)
	mux.HandleFunc("/api/v1/profiles/import", s.handleProfilesImport)
	mux.HandleFunc("/api/v1/profiles/promote-reference-draft", s.handleProfilesPromoteReferenceDraft)
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

	return withCORS(withDashboardAuth(withMutationAuth(mux)))
}

// withCORS keeps same-origin production access closed while allowing explicit
// development origins when MAVLINK_ANYWHERE_CORS_ALLOWED_ORIGIN is configured.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyCORSHeaders(w, r)
		if r.Method == "OPTIONS" {
			if r.Header.Get("Origin") != "" && w.Header().Get("Access-Control-Allow-Origin") == "" {
				writeError(w, http.StatusForbidden, "CORS origin not allowed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func applyCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return
	}
	allowed := strings.TrimSpace(os.Getenv(corsAllowedOriginEnv))
	if allowed == "" {
		return
	}
	for _, candidate := range strings.Split(allowed, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if candidate == origin || (candidate == "*" && isLoopbackRemote(r.RemoteAddr)) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Mavlink-Anywhere-Token, "+dashboardCSRFHeader)
			return
		}
	}
}

func withMutationAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodOptions || !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			h.ServeHTTP(w, r)
			return
		}
		if mutationAccessAllowed(r) {
			h.ServeHTTP(w, r)
			return
		}
		if dashboardBasicAuthAllowed(r) && !dashboardBrowserMutationAllowed(r) {
			writeError(w, http.StatusForbidden, dashboardCSRFHeader+" is required for browser mutating requests")
			return
		}
		if dashboardAuthConfigured() {
			w.Header().Set("WWW-Authenticate", dashboardAuthBasicRealm)
			writeError(w, http.StatusUnauthorized, "dashboard authentication required")
			return
		}
		if strings.TrimSpace(os.Getenv(mutationTokenEnv)) != "" {
			writeError(w, http.StatusForbidden, "invalid MAVLink Anywhere mutation token")
			return
		}
		writeError(w, http.StatusForbidden, mutationTokenEnv+" is required for remote mutating requests")
	})
}

func mutationAccessAllowed(r *http.Request) bool {
	if isLoopbackRemote(r.RemoteAddr) {
		return true
	}
	if dashboardBasicAuthAllowed(r) {
		return dashboardBrowserMutationAllowed(r)
	}
	if !dashboardAuthConfigured() &&
		strings.TrimSpace(os.Getenv(mutationTokenEnv)) == "" &&
		envFlagEnabled(openMutationsEnv) {
		return true
	}
	expected := strings.TrimSpace(os.Getenv(mutationTokenEnv))
	if expected == "" {
		return false
	}
	supplied := strings.TrimSpace(r.Header.Get("X-Mavlink-Anywhere-Token"))
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if supplied == "" && strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		supplied = strings.TrimSpace(auth[7:])
	}
	return supplied != "" && subtle.ConstantTimeCompare([]byte(supplied), []byte(expected)) == 1
}

func dashboardBrowserMutationAllowed(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	return strings.TrimSpace(r.Header.Get(dashboardCSRFHeader)) != ""
}

func envFlagEnabled(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func dashboardAuthConfigured() bool {
	return strings.TrimSpace(os.Getenv(dashboardAuthUserEnv)) != "" &&
		strings.TrimSpace(os.Getenv(dashboardAuthBcryptEnv)) != ""
}

func dashboardBasicAuthAllowed(r *http.Request) bool {
	expectedUser := strings.TrimSpace(os.Getenv(dashboardAuthUserEnv))
	expectedHash := strings.TrimSpace(os.Getenv(dashboardAuthBcryptEnv))
	if expectedUser == "" || expectedHash == "" {
		return false
	}
	user, password, ok := r.BasicAuth()
	if !ok {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(user), []byte(expectedUser)) != 1 {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(expectedHash), []byte(password)) == nil
}

func withDashboardAuth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions ||
			!dashboardAuthConfigured() ||
			isLoopbackRemote(r.RemoteAddr) ||
			mutationAccessAllowed(r) ||
			dashboardBasicAuthAllowed(r) {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("WWW-Authenticate", dashboardAuthBasicRealm)
		writeError(w, http.StatusUnauthorized, "dashboard authentication required")
	})
}

func isLoopbackRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
