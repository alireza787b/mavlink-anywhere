package config

import (
	"fmt"
	"strings"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/endpoints"
)

// ValidateEndpointTopology ensures the candidate does not create local bind
// conflicts. UDP normal-mode endpoints are outbound only and are allowed to
// share the same remote port as a local server-mode endpoint.
func ValidateEndpointTopology(existing []endpoints.Endpoint, candidate endpoints.Endpoint, replaceName string) error {
	if candidate.Type != "UdpEndpoint" {
		return nil
	}

	mode := strings.ToLower(strings.TrimSpace(candidate.Mode))
	if mode == "" {
		mode = "normal"
	}
	if err := ValidateEndpointMode(mode); err != nil {
		return err
	}

	if mode != "server" {
		return nil
	}

	addr := candidate.Address
	if addr == "" {
		addr = "0.0.0.0"
	}

	for _, ep := range existing {
		if ep.Name == replaceName || ep.Type != "UdpEndpoint" || !ep.Enabled {
			continue
		}
		if strings.ToLower(strings.TrimSpace(ep.Mode)) != "server" {
			continue
		}
		otherAddr := ep.Address
		if otherAddr == "" {
			otherAddr = "0.0.0.0"
		}
		if bindAddressesConflict(otherAddr, addr) && ep.Port == candidate.Port {
			return fmt.Errorf("server-mode endpoint %q already binds %s:%d", ep.Name, addr, candidate.Port)
		}
	}

	return nil
}

func bindAddressesConflict(existing, candidate string) bool {
	existing = normalizeBindAddress(existing)
	candidate = normalizeBindAddress(candidate)
	if existing == candidate {
		return true
	}
	return existing == "0.0.0.0" || candidate == "0.0.0.0" || existing == "::" || candidate == "::"
}

func normalizeBindAddress(addr string) string {
	addr = strings.ToLower(strings.TrimSpace(addr))
	if addr == "" {
		return "0.0.0.0"
	}
	if addr == "localhost" {
		return "127.0.0.1"
	}
	return addr
}
