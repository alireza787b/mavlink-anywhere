# MAVLink-Anywhere CLI Reference

Complete reference for the mavlink-anywhere command-line interface.

## Overview

The `mavlink-anywhere` CLI provides a unified interface for installing, configuring, and managing mavlink-router.

```bash
mavlink-anywhere <command> [options]
```

## Commands

### install

Install mavlink-router from source.

```bash
sudo mavlink-anywhere install [OPTIONS]
```

**Options:**
| Option | Description |
|--------|-------------|
| `--skip-swap` | Skip swap space management during compilation |
| `--force, -f` | Force reinstallation even if already installed |
| `-h, --help` | Show help |

**Examples:**
```bash
# Standard installation
sudo mavlink-anywhere install

# Force reinstall
sudo mavlink-anywhere install --force

# Skip swap management (if you have enough RAM)
sudo mavlink-anywhere install --skip-swap
```

**Notes:**
- Requires root privileges
- Compilation may take 5-15 minutes depending on hardware
- Automatically manages swap space during compilation
- Safe to run multiple times (idempotent)

---

### configure

Configure mavlink-router with UART, UDP endpoints, and service setup.

```bash
sudo mavlink-anywhere configure [OPTIONS]
```

This command is an alias for `./configure_mavlink_router.sh`.

**Modes:**

| Mode | Description |
|------|-------------|
| (default) | Interactive mode with prompts |
| `--auto` | Auto-detect settings, minimal prompts |
| `--headless` | No prompts, all settings via CLI |

**Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `--uart DEVICE` | UART device path | Auto-detected |
| `--baud RATE` | Baud rate | 57600 |
| `--endpoints LIST` | Comma-separated endpoints | - |
| `--gcs-ip IP` | GCS IP address (adds :24550) | - |
| `--input-type TYPE` | Input type: `uart` or `udp` | uart |
| `--input-address ADDR` | UDP listen address | 0.0.0.0 |
| `--input-port PORT` | UDP listen port | 14550 |
| `--debug` | Enable debug output | - |

**Examples:**

```bash
# Interactive mode (original behavior)
sudo mavlink-anywhere configure

# Auto mode with GCS IP
sudo mavlink-anywhere configure --auto --gcs-ip 192.168.1.100

# Headless with full control
sudo mavlink-anywhere configure --headless \
    --uart /dev/ttyS0 \
    --baud 57600 \
    --endpoints "127.0.0.1:14540,127.0.0.1:14569,192.168.1.100:24550"

# UDP input for SITL
sudo mavlink-anywhere configure --headless \
    --input-type udp \
    --input-port 14550 \
    --endpoints "127.0.0.1:14540"
```

---

### status

Show current mavlink-router status and configuration.

```bash
mavlink-anywhere status
```

**Output includes:**
- Binary installation status
- Service status (running/stopped/enabled)
- Serial port configuration status
- Current configuration summary

**Example output:**
```
┌──────────────────────────────────────────────────────────────────────────────┐
│  MAVLink Router Service Status                                                │
├──────────────────────────────────────────────────────────────────────────────┤
│ Binary:      ✓ Installed (mavlink-router 2)                                   │
│ Service:     ✓ Configured                                                     │
│ Status:      ● Running                                                        │
│ Since:       2024-01-15 10:30:00                                              │
│ Config:      ✓ /etc/mavlink-router/main.conf                                  │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### test

Test serial/UART connection for MAVLink data.

```bash
mavlink-anywhere test [OPTIONS]
```

**Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `--uart, --device DEVICE` | UART device to test | Auto-detected |
| `--baud RATE` | Baud rate | 57600 |

**Examples:**
```bash
# Test auto-detected device
mavlink-anywhere test

# Test specific device
mavlink-anywhere test --uart /dev/ttyAMA0

# Test with specific baud rate
mavlink-anywhere test --uart /dev/ttyS0 --baud 115200
```

**Output:**
- Device accessibility check
- Permission verification
- Data reception test (5 second timeout)
- MAVLink data detection

---

### start

Start the mavlink-router service.

```bash
sudo mavlink-anywhere start
```

**Notes:**
- Requires root privileges
- Service must be configured first
- Verifies service started successfully

---

### stop

Stop the mavlink-router service.

```bash
sudo mavlink-anywhere stop
```

---

### restart

Restart the mavlink-router service.

```bash
sudo mavlink-anywhere restart
```

---

### logs

Show mavlink-router service logs.

```bash
mavlink-anywhere logs [OPTIONS]
```

**Options:**

| Option | Description | Default |
|--------|-------------|---------|
| `-f, --follow` | Follow log output in real-time | - |
| `-n, --lines N` | Show last N lines | 50 |

**Examples:**
```bash
# Show recent logs
mavlink-anywhere logs

# Show last 100 lines
mavlink-anywhere logs -n 100

# Follow logs in real-time
mavlink-anywhere logs -f
```

---

### uninstall

Remove mavlink-router service configuration.

```bash
sudo mavlink-anywhere uninstall [OPTIONS]
```

**Options:**

| Option | Description |
|--------|-------------|
| `--remove-config` | Also remove configuration files |

**Examples:**
```bash
# Remove service only
sudo mavlink-anywhere uninstall

# Remove service and configuration
sudo mavlink-anywhere uninstall --remove-config
```

**Notes:**
- This removes the systemd service but not the mavlink-routerd binary
- Configuration files at `/etc/mavlink-router/` are kept unless `--remove-config` is specified

---

### help

Show help message.

```bash
mavlink-anywhere help
```

---

### version

Show version information.

```bash
mavlink-anywhere version
```

---

## Endpoint Format

Endpoints are specified as `IP:PORT` pairs, comma-separated:

```
"127.0.0.1:14540,127.0.0.1:14569,192.168.1.100:24550"
```

### Standard MDS Endpoints

| Port | Service | Description |
|------|---------|-------------|
| 14540 | MAVSDK | MAVSDK SDK connection |
| 14569 | mavlink2rest | Web-based REST API |
| 12550 | Local | Local monitoring/debugging |
| 24550 | GCS (VPN) | Remote ground station over VPN |
| 14550 | Standard | Standard MAVLink port |

### Named Shortcuts (in config files)

The following shortcuts are supported:
- `mavsdk` → 127.0.0.1:14540
- `mavlink2rest` → 127.0.0.1:14569
- `local` → 127.0.0.1:12550

---

## Configuration Files

| File | Description |
|------|-------------|
| `/etc/mavlink-router/main.conf` | Main mavlink-router configuration |
| `/etc/default/mavlink-router` | Environment variables |
| `/etc/systemd/system/mavlink-router.service` | Systemd service file |

---

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Permission denied (need sudo) |
| 3 | Configuration error |
| 4 | Service error |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `MA_DEBUG` | Enable debug output (`true`/`false`) |

---

## See Also

- [UART-SETUP.md](UART-SETUP.md) - Raspberry Pi serial configuration
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues and solutions
- [Main README](../README.md) - Project overview
