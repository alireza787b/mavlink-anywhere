# MAVLink-Anywhere Web Dashboard

The web dashboard provides a browser-based interface for monitoring and managing mavlink-router — no SSH or terminal knowledge required.

## Quick Access

After running `configure_mavlink_router.sh`, the dashboard is automatically installed and bound to localhost by default:

```
http://127.0.0.1:9070
```

For network access, expose it explicitly:

```bash
sudo ./configure_mavlink_router.sh --install-dashboard \
  --dashboard-listen 0.0.0.0:9070
```

## Features

- **Service Status** — See if mavlink-router is running, uptime, and version at a glance
- **Endpoint Management** — View, add, edit, delete, and toggle MAVLink endpoints
- **Guided Add Wizard** — Step-by-step wizard for adding new endpoints (GCS, local services, VPN)
- **Live Logs** — Real-time streaming of mavlink-router logs via SSE
- **Service Control** — Start, stop, restart mavlink-router from the browser
- **System Info** — Board detection, UART status, firewall info, RAM usage
- **Raw Config Editor** — Advanced users can edit the INI config directly
- **Default Server Endpoint** — Port 14550 listens for any GCS connection out of the box

## Architecture

The dashboard is a single static Go binary (~7MB) with the web UI embedded. It:

- Reads/writes the same `/etc/mavlink-router/main.conf` as CLI scripts
- Uses systemctl for service control
- Streams logs via `journalctl`
- Runs as a separate systemd service (`mavlink-anywhere-dashboard`)
- Uses <20MB RAM (hard-capped at 30MB via systemd MemoryMax)

## Installation

### Automatic (default)

The dashboard is installed automatically when you run:

```bash
sudo ./configure_mavlink_router.sh
```

### Skip Dashboard

```bash
sudo ./configure_mavlink_router.sh --skip-dashboard
```

### Install/Update Dashboard Only

```bash
sudo ./configure_mavlink_router.sh --install-dashboard
```

### Install/Update Dashboard and Expose on Network

```bash
sudo ./configure_mavlink_router.sh --install-dashboard \
  --dashboard-listen 0.0.0.0:9070
```

### Manual

Download the binary for your architecture from [GitHub Releases](https://github.com/alireza787b/mavlink-anywhere/releases):

```bash
# Raspberry Pi Zero/W (armv6)
curl -fsSL https://github.com/alireza787b/mavlink-anywhere/releases/latest/download/mavlink-anywhere-linux-arm6 \
  -o /opt/mavlink-anywhere/mavlink-anywhere
chmod +x /opt/mavlink-anywhere/mavlink-anywhere

# Raspberry Pi 3/4/5, Jetson (arm64)
curl -fsSL https://github.com/alireza787b/mavlink-anywhere/releases/latest/download/mavlink-anywhere-linux-arm64 \
  -o /opt/mavlink-anywhere/mavlink-anywhere
chmod +x /opt/mavlink-anywhere/mavlink-anywhere

# x86_64
curl -fsSL https://github.com/alireza787b/mavlink-anywhere/releases/latest/download/mavlink-anywhere-linux-amd64 \
  -o /opt/mavlink-anywhere/mavlink-anywhere
chmod +x /opt/mavlink-anywhere/mavlink-anywhere
```

Then start manually:

```bash
./mavlink-anywhere --listen 127.0.0.1:9070
```

## Systemd Service

The dashboard runs as `mavlink-anywhere-dashboard.service`:

```bash
# Status
sudo systemctl status mavlink-anywhere-dashboard

# Restart
sudo systemctl restart mavlink-anywhere-dashboard

# Stop (does NOT affect mavlink-router)
sudo systemctl stop mavlink-anywhere-dashboard

# Disable at boot
sudo systemctl disable mavlink-anywhere-dashboard
```

## Security

| Access | Auth Required | Rationale |
|--------|--------------|-----------|
| `127.0.0.1:9070` (default) | None | Same trust level as SSH |
| `0.0.0.0:9070` (explicit) | None | Use only on trusted networks, VPNs, or behind SSH tunneling |

The dashboard binds to `127.0.0.1` by default — it's only accessible from the device itself or via SSH tunnel. To expose it on the network:

```bash
sudo ./configure_mavlink_router.sh --install-dashboard \
  --dashboard-listen 0.0.0.0:9070
```

## GCS Server Endpoint (Port 14550)

As of v3.0.0, mavlink-anywhere includes a default **server-mode** endpoint on port 14550. This means:

- Any GCS (QGroundControl, Mission Planner) can connect TO the device by pointing at its IP:14550
- No pre-configuration of GCS IP is needed on the device
- Works out of the box for ad-hoc connections

**QGroundControl**: Comm Links → Add → UDP → Server: `<device-ip>` → Port: `14550`

## API Reference

The dashboard exposes a REST API for programmatic access:

```
GET    /api/v1/status              # Service status, version, board info
GET    /api/v1/config              # Current config as JSON
PUT    /api/v1/config              # Write raw config
GET    /api/v1/endpoints           # List all endpoints
POST   /api/v1/endpoints           # Add endpoint
PUT    /api/v1/endpoints/{name}    # Update endpoint
DELETE /api/v1/endpoints/{name}    # Delete endpoint
PATCH  /api/v1/endpoints/{name}    # Toggle enable/disable
GET    /api/v1/input               # Current input source
PUT    /api/v1/input               # Change input source
POST   /api/v1/service/restart     # Restart mavlink-router
POST   /api/v1/service/stop        # Stop mavlink-router
POST   /api/v1/service/start       # Start mavlink-router
GET    /api/v1/logs/stream         # SSE real-time log stream
GET    /api/v1/logs/recent?n=100   # Last N log lines
GET    /api/v1/system/info         # Board and firewall info
POST   /api/v1/system/firewall     # Open a firewall port
GET    /api/v1/templates           # Endpoint templates for wizard
GET    /api/v1/health              # Health check
```

## Cross-Platform Support

| Platform | Architecture | Binary | Status |
|----------|-------------|--------|--------|
| Raspberry Pi Zero/W | armv6 | `linux-arm6` | Full support |
| Raspberry Pi 3/4/5 | arm64 | `linux-arm64` | Full support |
| NVIDIA Jetson | arm64 | `linux-arm64` | Full support |
| Ubuntu/Debian x86 | amd64 | `linux-amd64` | Full support |

## Troubleshooting

**Dashboard not accessible:**
```bash
# Check if service is running
sudo systemctl status mavlink-anywhere-dashboard

# Check what port it's listening on
ss -tlnp | grep 9070

# View dashboard logs
sudo journalctl -u mavlink-anywhere-dashboard -f
```

If the service is bound to localhost only, use SSH tunneling:

```bash
ssh -L 9070:localhost:9070 user@<device-ip>
# Then open http://127.0.0.1:9070 locally
```

**Dashboard shows "No endpoints":**
- Verify config exists: `cat /etc/mavlink-router/main.conf`
- Re-run: `sudo ./configure_mavlink_router.sh`
