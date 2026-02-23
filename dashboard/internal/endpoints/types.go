package endpoints

// Endpoint represents a mavlink-router endpoint parsed from config.
type Endpoint struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // "UdpEndpoint", "UartEndpoint", "TcpEndpoint"
	Mode        string `json:"mode"`        // "normal", "server"
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Device      string `json:"device,omitempty"`
	Baud        int    `json:"baud,omitempty"`
	Description string `json:"description"`
	Category    string `json:"category"` // "gcs", "local", "input", "custom"
	Removable   bool   `json:"removable"`
	Enabled     bool   `json:"enabled"`
}

// EndpointTemplate defines a guided-add template.
type EndpointTemplate struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Category    string `json:"category"`
	Mode        string `json:"mode"`
	DefaultAddr string `json:"defaultAddress"`
	DefaultPort int    `json:"defaultPort"`
	Description string `json:"description"`
	InfoText    string `json:"infoText"`
}
