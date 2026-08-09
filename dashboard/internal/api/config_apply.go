package api

import (
	"fmt"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/profiles"
	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/system"
)

// applyRouterMutation gives every dashboard config change the same behavior:
// backup first, mutate, restart only when already running, and roll back if the
// change cannot be applied.
func (s *Server) applyRouterMutation(mutate func() error) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := profiles.CreateBackup(s.configPath, s.envPath); err != nil {
		return false, fmt.Errorf("create backup: %w", err)
	}
	wasRunning := system.GetServiceStatus().State == "running"
	if err := mutate(); err != nil {
		_, _ = profiles.RestoreLatest(s.configPath, s.envPath)
		return false, err
	}
	if !wasRunning {
		return false, nil
	}
	if err := system.RestartService(); err != nil {
		if _, rollbackErr := profiles.RestoreLatest(s.configPath, s.envPath); rollbackErr != nil {
			return false, fmt.Errorf("restart failed and rollback failed: %v", rollbackErr)
		}
		if rollbackRestartErr := system.RestartService(); rollbackRestartErr != nil {
			return false, fmt.Errorf("restart failed; previous config was restored but its restart also failed")
		}
		return false, fmt.Errorf("restart failed; previous config was restored")
	}
	return true, nil
}
