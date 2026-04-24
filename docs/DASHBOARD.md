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
- **Routing View** — Input source is separated from listener and output routes
- **Endpoint Management** — View, add, edit, delete, and toggle MAVLink endpoints
- **MAVLink Health** — Passive runtime probe of the routed stream (bytes, packets, heartbeat, system IDs)
- **Warnings & Guidance** — Detect missing input, firewall issues, duplicate server binds, and mixed routing patterns
- **Guided Add Wizard** — Step-by-step wizard for adding new endpoints (GCS, local services, VPN)
- **Live Logs** — Real-time streaming of mavlink-router logs via SSE
- **Service Control** — Start, stop, restart mavlink-router from the browser
- **System Info** — Board detection, UART status, firewall info, RAM usage
- **Routing Profiles** — Export the current effective routing profile, preview an import, apply with backup, and restore the last good backup
- **Raw Config Editor** — Advanced users can edit the INI config directly
- **Default Server Endpoint** — Port 14550 listens for any GCS connection out of the box
- **Source Visibility** — The active MAVLink source is shown separately from output endpoints

## Architecture

The dashboard is a single static Go binary (~7MB) with the web UI embedded. It:

- Reads/writes the same `/etc/mavlink-router/main.conf` as CLI scripts
- Uses systemctl for service control
- Streams logs via `journalctl`
- Runs as a separate systemd service (`mavlink-anywhere-dashboard`)
- Uses <20MB RAM (hard-capped at 30MB via systemd MemoryMax)
- Works without Node/npm at runtime

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

On supported release architectures (`arm6`, `arm64`, `amd64`), the installer downloads a prebuilt binary. If that download is unavailable and `go` is installed locally, the installer falls back to a local source build. If neither path succeeds, mavlink-router still installs and the dashboard is skipped. Minimal hosts do not need the external `file(1)` package for the download validation step.

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

### Manual Source Build

If your architecture does not have a published release asset, build the dashboard directly:

```bash
cd dashboard
CGO_ENABLED=0 go build -o ../mavlink-anywhere ./cmd/
sudo install -m 755 ../mavlink-anywhere /opt/mavlink-anywhere/mavlink-anywhere
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
- Works out of the box for ad-hoc connections after the GCS sends first

Important routing notes:

- `gcs_listen` is best for ad-hoc field access, not deterministic multi-client fanout
- UDP server mode effectively tracks the last active sender on that endpoint
- Local consumers such as MAVSDK (`127.0.0.1:14540`) and mavlink2rest (`127.0.0.1:14569`) should remain explicit normal-mode endpoints
- An explicit outbound endpoint to a remote `:14550` can coexist with `gcs_listen` without a local bind conflict
- The same remote GCS should not consume both paths at once, or it may see duplicate telemetry
- For multiple dynamic remote clients, prefer the default TCP server on `5760`

This uses `mavlink-router` UDP **server mode**, so the router sends replies to the IP:port of the **last client that sent traffic** on that endpoint. Treat it as a convenient device-side listener, not as a multi-client fanout bus.

If you need multiple simultaneous remote consumers:

- Add dedicated `Mode=Normal` endpoints for each remote IP:port
- Or use the default TCP server on port `5760`

**QGroundControl**: Comm Links → Add → UDP → Server: `<device-ip>` → Port: `14550`

If you delete `gcs_listen`, use **Add Endpoint** and choose **Listen for GCS** to restore it.

## TCP Server (Port 5760)

`mavlink-router` listens on `5760/tcp` by default. Any TCP client connecting there can send and receive routed MAVLink data.

Use this when you want:

- multiple dynamic clients without predefining every remote IP
- a clean TCP path for tools that prefer `tcp://` connections
- a lightweight read-only health probe from the dashboard

PX4 SITL does not reserve `5760` by default. PX4's documented simulator defaults are UDP `14550`/`14540` plus simulator TCP `4560`. A conflict only exists if your own SITL stack or another service explicitly binds `5760`.

## Update Workflow

Update the installed tool and dashboard:

```bash
cd ~/mavlink-anywhere
git fetch --tags origin
git pull --ff-only
sudo ./configure_mavlink_router.sh --install-dashboard
```

`--install-dashboard` updates the dashboard binary when the installed version is older than the checked-out `mavlink-anywhere` release.

If you also want to refresh the installed `mavlink-routerd` binary from source:

```bash
cd ~/mavlink-anywhere
sudo ./install_mavlink_router.sh
```

If the dashboard is exposed on the network, keep the explicit bind:

```bash
sudo ./configure_mavlink_router.sh --install-dashboard \
  --dashboard-listen 0.0.0.0:9070
```

## API Reference

The dashboard exposes a REST API for programmatic access:

```
GET    /api/v1/status              # Service status, version, board info
GET    /api/v1/diagnostics         # MAVLink probe, warnings, docs links
GET    /api/v1/config              # Current config as JSON
PUT    /api/v1/config              # Write raw config
GET    /api/v1/profiles/export     # Export current effective routing profile
POST   /api/v1/profiles/preview    # Validate and preview imported profile
POST   /api/v1/profiles/apply      # Apply imported profile with backup
GET    /api/v1/profiles/backups    # List routing backups
POST   /api/v1/profiles/restore    # Restore the latest backup
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

If your Linux architecture is outside this list, use CLI-only mode or the manual source build above.

## Routing Profiles

Routing profiles are a dashboard-first feature for repeatable endpoint layouts.

What gets exported:

- router general settings
- all configured endpoints, including the input endpoint
- lightweight metadata such as profile name and export timestamp

What does **not** get exported:

- logs
- runtime process state
- firewall rules
- host bootloader serial settings

Import workflow:

1. choose a `.json` profile file
2. preview the changes
3. choose `Replace current routing` or `Merge named endpoints`
4. apply
5. dashboard creates a backup and restarts `mavlink-router`

Current rules:

- importing a profile does **not** require reboot
- reboot is only required when you separately change host serial boot settings
- replace mode removes endpoints that are not present in the imported profile
- merge mode keeps existing endpoints unless an imported endpoint with the same name replaces them
- the dashboard regenerates `/etc/default/mavlink-router` from the effective config so CLI, docs, and UI stay aligned

Restore workflow:

- `Restore Last Good` restores the latest dashboard-created backup
- it writes both the config file and the companion env file
- it restarts `mavlink-router` after restore

## Not Yet Implemented

- Token-based auth for non-local dashboard exposure
- Deep FC sensor/firmware discovery beyond passive routed-stream detection

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
