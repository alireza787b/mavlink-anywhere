package mavlink

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// Health captures a short runtime probe of the MAVLink stream.
type Health struct {
	Available      bool   `json:"available"`
	Active         bool   `json:"active"`
	Source         string `json:"source,omitempty"`
	TCPServerPort  int    `json:"tcpServerPort"`
	BytesReceived  int    `json:"bytesReceived"`
	ParsedPackets  int    `json:"parsedPackets"`
	Heartbeats     int    `json:"heartbeats"`
	SampleWindowMs int    `json:"sampleWindowMs"`
	SystemID       int    `json:"systemId,omitempty"`
	ComponentID    int    `json:"componentId,omitempty"`
	VehicleType    string `json:"vehicleType,omitempty"`
	Autopilot      string `json:"autopilot,omitempty"`
	SystemStatus   string `json:"systemStatus,omitempty"`
	MAVLinkVersion int    `json:"mavlinkVersion,omitempty"`
	Note           string `json:"note,omitempty"`
	Error          string `json:"error,omitempty"`
}

// ProbeTCP opens a short-lived connection to mavlink-router's TCP server and
// samples the routed stream. This is intentionally lightweight and read-only.
func ProbeTCP(port int, timeout time.Duration) Health {
	health := Health{
		Available:      false,
		TCPServerPort:  port,
		SampleWindowMs: int(timeout.Milliseconds()),
	}
	if port <= 0 {
		health.Error = "TCP server is disabled"
		return health
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		health.Error = "Unable to connect to mavlink-router TCP server"
		return health
	}
	defer conn.Close()

	health.Available = true
	_ = conn.SetDeadline(time.Now().Add(timeout))

	buf := make([]byte, 4096)
	stream := make([]byte, 0, 8192)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			stream = append(stream, buf[:n]...)
			health.BytesReceived += n
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			health.Error = err.Error()
			break
		}
	}

	parsed := parseStream(stream)
	health.ParsedPackets = parsed.Packets
	health.Heartbeats = parsed.Heartbeats
	health.Active = parsed.Packets > 0

	if parsed.HeartbeatSeen {
		health.SystemID = int(parsed.SystemID)
		health.ComponentID = int(parsed.ComponentID)
		health.VehicleType = vehicleTypeName(parsed.VehicleType)
		health.Autopilot = autopilotName(parsed.Autopilot)
		health.SystemStatus = systemStatusName(parsed.SystemStatus)
		health.MAVLinkVersion = int(parsed.MAVLinkVersion)
		health.Note = "Heartbeat detected on routed MAVLink stream"
		return health
	}

	if health.BytesReceived > 0 {
		health.Note = "Received MAVLink bytes but no heartbeat was seen during the sample window"
	} else if health.Error == "" {
		health.Note = "Connected to mavlink-router, but no MAVLink data arrived during the sample window"
	}

	return health
}

type parsedStats struct {
	Packets        int
	Heartbeats     int
	HeartbeatSeen  bool
	SystemID       byte
	ComponentID    byte
	VehicleType    byte
	Autopilot      byte
	SystemStatus   byte
	MAVLinkVersion byte
}

func parseStream(data []byte) parsedStats {
	var stats parsedStats
	for i := 0; i < len(data); {
		switch data[i] {
		case 0xFE:
			total, hb, ok := parseV1Frame(data[i:], &stats)
			if !ok {
				i++
				continue
			}
			stats.Packets++
			if hb {
				stats.Heartbeats++
			}
			i += total
		case 0xFD:
			total, hb, ok := parseV2Frame(data[i:], &stats)
			if !ok {
				i++
				continue
			}
			stats.Packets++
			if hb {
				stats.Heartbeats++
			}
			i += total
		default:
			i++
		}
	}
	return stats
}

func parseV1Frame(data []byte, stats *parsedStats) (int, bool, bool) {
	if len(data) < 8 {
		return 0, false, false
	}
	payloadLen := int(data[1])
	total := payloadLen + 8
	if len(data) < total {
		return 0, false, false
	}
	msgID := data[5]
	if msgID != 0 || payloadLen < 9 {
		return total, false, true
	}
	payload := data[6 : 6+payloadLen]
	stats.HeartbeatSeen = true
	stats.SystemID = data[3]
	stats.ComponentID = data[4]
	stats.VehicleType = payload[4]
	stats.Autopilot = payload[5]
	stats.SystemStatus = payload[7]
	stats.MAVLinkVersion = payload[8]
	return total, true, true
}

func parseV2Frame(data []byte, stats *parsedStats) (int, bool, bool) {
	if len(data) < 12 {
		return 0, false, false
	}
	payloadLen := int(data[1])
	incompatFlags := data[2]
	total := payloadLen + 12
	if incompatFlags&0x01 != 0 {
		total += 13
	}
	if len(data) < total {
		return 0, false, false
	}
	msgID := uint32(data[7]) | uint32(data[8])<<8 | uint32(data[9])<<16
	if msgID != 0 || payloadLen < 9 {
		return total, false, true
	}
	payload := data[10 : 10+payloadLen]
	stats.HeartbeatSeen = true
	stats.SystemID = data[5]
	stats.ComponentID = data[6]
	stats.VehicleType = payload[4]
	stats.Autopilot = payload[5]
	stats.SystemStatus = payload[7]
	stats.MAVLinkVersion = payload[8]
	return total, true, true
}

func autopilotName(v byte) string {
	switch v {
	case 3:
		return "ArduPilot"
	case 12:
		return "PX4"
	default:
		return fmt.Sprintf("Autopilot %d", v)
	}
}

func vehicleTypeName(v byte) string {
	switch v {
	case 1:
		return "Fixed Wing"
	case 2:
		return "Quadrotor"
	case 6:
		return "Ground Rover"
	case 10:
		return "VTOL"
	case 13:
		return "Hexarotor"
	case 14:
		return "Octorotor"
	default:
		return fmt.Sprintf("Type %d", v)
	}
}

func systemStatusName(v byte) string {
	switch v {
	case 3:
		return "Standby"
	case 4:
		return "Active"
	case 5:
		return "Critical"
	case 6:
		return "Emergency"
	case 7:
		return "Power Off"
	default:
		return fmt.Sprintf("State %d", v)
	}
}

// heartbeatPayloadCustomMode is intentionally retained for future profile or UI
// expansions where mode bits may be surfaced.
func heartbeatPayloadCustomMode(payload []byte) uint32 {
	if len(payload) < 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(payload[:4])
}
