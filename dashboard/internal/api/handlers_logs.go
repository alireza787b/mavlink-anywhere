package api

import (
	"bufio"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
)

func (s *Server) handleLogStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	cmd := exec.Command("journalctl", "-u", "mavlink-router", "-f", "--no-pager", "-o", "short-iso")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start log stream")
		return
	}

	if err := cmd.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start journalctl: "+err.Error())
		return
	}

	// Clean up on disconnect
	go func() {
		<-r.Context().Done()
		cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}

	cmd.Wait()
}

func (s *Server) handleLogsRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "GET only")
		return
	}

	n := 100
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, err := strconv.Atoi(nStr); err == nil && v > 0 && v <= 1000 {
			n = v
		}
	}

	cmd := exec.Command("journalctl", "-u", "mavlink-router", "-n", strconv.Itoa(n), "--no-pager", "-o", "short-iso")
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"lines": []string{},
			"error": "Failed to read logs: " + err.Error(),
		})
		return
	}

	lines := []string{}
	for _, line := range splitLines(string(out)) {
		if line != "" {
			lines = append(lines, line)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"lines": lines,
	})
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
