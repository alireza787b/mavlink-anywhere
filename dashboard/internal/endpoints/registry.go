package endpoints

// Registry is the built-in list of endpoint templates for the guided-add wizard.
var Registry = []EndpointTemplate{
	{
		ID:          "gcs_push",
		Label:       "Send to GCS",
		Category:    "gcs",
		Mode:        "normal",
		DefaultAddr: "192.168.1.100",
		DefaultPort: 14550,
		Description: "Push MAVLink data to a specific ground station IP",
		InfoText:    "Sends telemetry to your ground station at the specified address. You must know the GCS IP.",
	},
	{
		ID:          "gcs_listen",
		Label:       "Listen for GCS",
		Category:    "gcs",
		Mode:        "server",
		DefaultAddr: "0.0.0.0",
		DefaultPort: 14550,
		Description: "Any GCS can connect to this device on this port",
		InfoText:    "GCS connects TO this device — no IP pre-configuration needed. Works like ARK OS.",
	},
	{
		ID:          "mavsdk",
		Label:       "MAVSDK",
		Category:    "local",
		Mode:        "normal",
		DefaultAddr: "127.0.0.1",
		DefaultPort: 14540,
		Description: "MAVSDK drone control SDK connection",
		InfoText:    "Forwards MAVLink to MAVSDK running on this device.",
	},
	{
		ID:          "mavlink2rest",
		Label:       "mavlink2rest",
		Category:    "local",
		Mode:        "normal",
		DefaultAddr: "127.0.0.1",
		DefaultPort: 14569,
		Description: "REST API for web-based MAVLink access",
		InfoText:    "Forwards MAVLink to mavlink2rest for HTTP API access.",
	},
	{
		ID:          "local",
		Label:       "Local telemetry",
		Category:    "local",
		Mode:        "normal",
		DefaultAddr: "127.0.0.1",
		DefaultPort: 12550,
		Description: "Local telemetry monitoring port",
		InfoText:    "For local services that need MAVLink data.",
	},
	{
		ID:          "gcs_vpn",
		Label:       "GCS over VPN",
		Category:    "gcs",
		Mode:        "normal",
		DefaultAddr: "100.96.0.1",
		DefaultPort: 24550,
		Description: "Remote GCS over VPN (Tailscale/ZeroTier)",
		InfoText:    "Push telemetry to a remote ground station over VPN tunnel.",
	},
	{
		ID:          "custom",
		Label:       "Custom endpoint",
		Category:    "custom",
		Mode:        "normal",
		DefaultAddr: "0.0.0.0",
		DefaultPort: 14550,
		Description: "Custom UDP endpoint with full control",
		InfoText:    "Advanced: specify all parameters manually.",
	},
}

// DescriptionForEndpoint returns a human-readable description
// for a parsed endpoint based on well-known ports/addresses.
func DescriptionForEndpoint(name, mode, address string, port int) string {
	// Check registry first by name match
	for _, t := range Registry {
		if t.ID == name {
			return t.Description
		}
	}

	// Infer from address/port
	if address == "127.0.0.1" || address == "localhost" {
		switch port {
		case 14540:
			return "MAVSDK drone control SDK connection"
		case 14569:
			return "REST API for web-based MAVLink access"
		case 12550:
			return "Local telemetry monitoring port"
		}
		return "Local service connection"
	}

	if mode == "server" {
		return "Listening for incoming GCS connections"
	}

	if port == 24550 {
		return "Remote GCS over VPN"
	}

	return "UDP endpoint to " + address
}

// CategoryForEndpoint infers the category of an endpoint.
func CategoryForEndpoint(name, mode, address string, port int) string {
	if address == "127.0.0.1" || address == "localhost" {
		return "local"
	}
	if mode == "server" {
		return "gcs"
	}
	return "gcs"
}
