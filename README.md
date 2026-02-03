# MAVLinkAnywhere

MAVLinkAnywhere is a general-purpose project that enables MAVLink data streaming to both local endpoints and remote locations over the internet. This project provides simplified scripts to install and configure `mavlink-router` on companion computers (Raspberry Pi, Jetson, etc.). `mavlink-router` is a powerful application that routes MAVLink packets between various endpoints, including UART, UDP, and TCP, making it ideal for MAVLink based UAV (PX4, Ardupilot, etc.) and drone communication systems.

[![MAVLinkAnywhere Tutorial](https://img.youtube.com/vi/_QEWpoy6HSo/0.jpg)](https://www.youtube.com/watch?v=_QEWpoy6HSo)

## Video Tutorial and Setup Guide

**New to mavlink-anywhere? Just follow the video!** The interactive setup is unchanged from v1.x.

Watch our comprehensive setup guide that walks you through the entire process:
- [Complete Guide to Stream Pixhawk/ArduPilot/PX4 Telemetry Data Anywhere (2024)](https://www.youtube.com/watch?v=_QEWpoy6HSo)

### Video Contents
- 00:00 - Introduction
- 02:15 - Setting up the Raspberry Pi
- 04:30 - Local MAVLINK Streaming
- 08:30 - Smart WiFi manager setup
- 11:40 - Internet-based MAVLink Streaming
- 15:00 - Outro

### Required Hardware
- Raspberry Pi (any model)
- Pixhawk/ArduPilot/PX4 flight controller
- Basic UART connection cables

## Prerequisites

Before starting with MAVLinkAnywhere, ensure that:
- Your companion computer (Raspberry Pi, Jetson, etc.) is installed with Ubuntu or Raspbian OS
- You have properly wired your Pixhawk's TELEM ports to the companion computer's UART TTL pins
- MAVLink streaming is enabled on the TELEM port of your flight controller

## Quick Start (Interactive Mode)

The classic setup method - perfect for beginners and compatible with the video tutorial:

```bash
# 1. Clone the repository
git clone https://github.com/alireza787b/mavlink-anywhere.git
cd mavlink-anywhere

# 2. Install mavlink-router
sudo ./install_mavlink_router.sh

# 3. Configure mavlink-router (interactive prompts)
sudo ./configure_mavlink_router.sh
```

## NEW: Automated/Headless Mode (v2.0)

For scripted deployments and automation (e.g., MDS integration):

### Auto Mode (Recommended for most users)
```bash
# Auto-detect UART, use standard endpoints, specify GCS IP
sudo ./configure_mavlink_router.sh --auto --gcs-ip 192.168.1.100
```

### Headless Mode (Full control)
```bash
# Complete control via CLI - no prompts
sudo ./configure_mavlink_router.sh --headless \
    --uart /dev/ttyS0 \
    --baud 57600 \
    --endpoints "127.0.0.1:14540,127.0.0.1:14569,192.168.1.100:24550"
```

### UDP Input Mode (For SITL/Simulation)
```bash
# Use UDP input instead of serial
sudo ./configure_mavlink_router.sh --headless \
    --input-type udp \
    --input-port 14550 \
    --endpoints "127.0.0.1:14540,127.0.0.1:14569"
```

## NEW: CLI Interface (v2.0)

The `mavlink-anywhere` CLI provides a unified interface:

```bash
# Show status
./mavlink-anywhere status

# Test serial connection
./mavlink-anywhere test

# View logs
./mavlink-anywhere logs -f

# See all commands
./mavlink-anywhere help
```

## Configuration Options

### Command Line Arguments

| Option | Description | Example |
|--------|-------------|---------|
| `--auto` | Auto-detect settings, minimal prompts | `--auto` |
| `--headless` | No prompts, all settings via CLI | `--headless` |
| `--uart DEVICE` | UART device path | `--uart /dev/ttyS0` |
| `--baud RATE` | Baud rate | `--baud 57600` |
| `--endpoints LIST` | Comma-separated endpoints | `--endpoints "127.0.0.1:14540,192.168.1.100:24550"` |
| `--gcs-ip IP` | GCS IP (adds :24550 endpoint) | `--gcs-ip 100.96.32.75` |
| `--input-type TYPE` | Input type: `uart` or `udp` | `--input-type udp` |
| `--input-port PORT` | UDP input port (for udp type) | `--input-port 14550` |

### Standard MDS Endpoints

When using `--auto` mode, these endpoints are configured automatically:

| Port | Service | Description |
|------|---------|-------------|
| 14540 | MAVSDK | Drone control SDK |
| 14569 | mavlink2rest | Web-based telemetry |
| 12550 | Local | Local monitoring |
| 24550 | GCS (VPN) | Remote ground station |

## Remote Connectivity

### Internet Connection Options
- **5G/4G/LTE**: Use USB Cellular dongles for mobile connectivity
- **Ethernet**: Direct connection to your network interface
- **WiFi**: For WiFi connectivity, we recommend using our [Smart WiFi Manager](https://github.com/alireza787b/smart-wifi-manager) project to ensure robust and reliable connections to your predefined networks
- **Satellite Internet**: Compatible with various satellite internet solutions

### VPN Solutions
For internet-based telemetry, you have several VPN options:
1. [NetBird](https://netbird.io/) (Recommended, demonstrated in video tutorial)
2. [WireGuard](https://www.wireguard.com/)
3. [Tailscale](https://tailscale.com/)
4. [ZeroTier](https://www.zerotier.com/)
   - [Legacy Setup Video from 2020](https://www.youtube.com/watch?v=WoRce4Re3Wg) (Note: Our 2024 method shown above is much simpler)

Alternatively, you can configure port forwarding on your router.

## What the Scripts Do

### Installation Script (`install_mavlink_router.sh`)
- Checks if `mavlink-router` is already installed
- Removes any existing `mavlink-router` directory
- Updates the system and installs required packages (`git`, `meson`, `ninja-build`, `pkg-config`, `gcc`, `g++`, `systemd`, `libsystemd-dev`, `python3-venv`)
- Intelligently manages swap space during compilation (supports both legacy `dphys-swapfile` and modern swap methods)
- Clones the `mavlink-router` repository and initializes its submodules
- Creates and activates a Python virtual environment
- Installs the Meson build system in the virtual environment
- Builds and installs `mavlink-router` using Meson and Ninja
- Resets the swap space to its original size after installation

### Configuration Script (`configure_mavlink_router.sh`)
- **Interactive mode**: Prompts for UART device, baud rate, and UDP endpoints
- **Auto mode**: Auto-detects UART, uses standard endpoints
- **Headless mode**: Accepts all settings via command line
- Creates environment file at `/etc/default/mavlink-router`
- Generates mavlink-router config at `/etc/mavlink-router/main.conf`
- Creates and enables systemd service for automatic startup

## Troubleshooting

### Build Issues

1. **"Dependency systemd not found"** - The script automatically handles this by explicitly setting the systemd directory path, bypassing pkg-config lookup issues common on some Debian Trixie systems.

2. **"dphys-swapfile: command not found"** - This is normal on newer systems. The script automatically falls back to standard Linux swap management.

3. **Build failures due to low memory** - The script automatically increases swap space to 2GB during compilation and restores it afterward.

### Serial Port Issues

1. **Device not found**: Ensure UART is enabled in `raspi-config` (Interface Options → Serial Port)

2. **Permission denied**: Add user to dialout group:
   ```bash
   sudo usermod -aG dialout $USER
   # Log out and back in for changes to take effect
   ```

3. **Serial console blocking UART**: Disable serial console in `raspi-config` (keep serial hardware enabled)

4. **Wrong device**: Check available devices:
   ```bash
   ls -la /dev/tty* /dev/serial*
   ```

### Service Issues

```bash
# Check service status
sudo systemctl status mavlink-router

# View detailed logs
sudo journalctl -u mavlink-router -f

# Restart service
sudo systemctl restart mavlink-router
```

## Integration with MDS (MAVSDK Drone Show)

mavlink-anywhere v2.0 is designed for seamless integration with [MDS](https://github.com/alireza787b/mavsdk_drone_show):

```bash
# MDS initialization script can now auto-configure mavlink-anywhere
sudo ./tools/mds_init.sh -d 1 -y --mavlink-auto --gcs-ip 100.96.32.75
```

See the [MDS documentation](https://github.com/alireza787b/mavsdk_drone_show) for more details.

## File Structure

```
mavlink-anywhere/
├── README.md                      # This file
├── install_mavlink_router.sh      # Installation script (unchanged)
├── configure_mavlink_router.sh    # Configuration script (enhanced)
├── mavlink-anywhere               # NEW: CLI entry point
├── lib/
│   ├── common.sh                  # Shared utilities
│   ├── detect.sh                  # Hardware detection
│   ├── config.sh                  # Configuration generation
│   └── service.sh                 # Systemd management
├── templates/
│   └── main.conf.template         # Configuration template
├── docs/
│   ├── CLI-REFERENCE.md           # CLI documentation
│   ├── UART-SETUP.md              # Serial setup guide
│   └── TROUBLESHOOTING.md         # Detailed troubleshooting
└── LICENSE                        # MIT License
```

## Monitoring and Logs
- **Check the status of the service:**
  ```sh
  sudo systemctl status mavlink-router
  ```
- **View detailed logs:**
  ```sh
  sudo journalctl -u mavlink-router -f
  ```

## Connecting with QGroundControl
Use QGroundControl to connect to your companion computer's IP address on the configured UDP endpoints. For internet-based telemetry, make sure to follow the setup video to properly register your devices on your chosen VPN system or configure port forwarding on your router.

## Contact
For more information, visit the [GitHub Repo](https://github.com/alireza787b/mavlink-anywhere).

## Related Resources
- [MAVSDK Drone Show (MDS)](https://github.com/alireza787b/mavsdk_drone_show)
- [Smart WiFi Manager Project](https://github.com/alireza787b/smart-wifi-manager)
- [NetBird VPN](https://netbird.io/)
- [Original 2020 Tutorial (Legacy Method)](https://www.youtube.com/watch?v=WoRce4Re3Wg)

## Support
If you encounter any issues, please:
1. Check the video tutorial timestamps for specific setup steps
2. Review the relevant sections in this documentation
3. Check [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) for detailed solutions
4. Open an issue on GitHub with detailed information about your setup

## License
MIT License - Copyright (c) 2024 Alireza Ghaderi

## Keywords
- MAVLink
- Raspberry Pi
- Drone Communication
- UAV
- mavlink-router
- UART
- UDP
- TCP
- QGroundControl
- Drone Telemetry
- Remote Telemetry
- VPN
- NetBird
- WireGuard
- Smart WiFi
- 4G Telemetry
- MDS
- MAVSDK Drone Show
