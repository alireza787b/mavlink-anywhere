# MAVLink Anywhere

**Stream MAVLink telemetry from your drone to anywhere in the world.**

Route MAVLink data from your flight controller (Pixhawk/ArduPilot/PX4) through a companion computer to ground stations, SDKs, and remote locations.

---

## 📺 Video Tutorial

**First time here? Watch the video!**

[![MAVLink Anywhere Tutorial](https://img.youtube.com/vi/_QEWpoy6HSo/0.jpg)](https://www.youtube.com/watch?v=_QEWpoy6HSo)

🎬 [Complete Setup Guide (YouTube)](https://www.youtube.com/watch?v=_QEWpoy6HSo)

---

## 🚀 Quick Start

### Step 1: Clone & Install

```bash
git clone https://github.com/alireza787b/mavlink-anywhere.git
cd mavlink-anywhere
sudo ./install_mavlink_router.sh
```

### Step 2: Configure

```bash
sudo ./configure_mavlink_router.sh
```

The script will:
- **Detect your platform** (Raspberry Pi, Jetson, generic Linux)
- **Check serial port** configuration
- **Guide you** through any needed setup (with auto-fix option on Raspberry Pi)
- **Configure** mavlink-router with your settings

> **Note:** On Raspberry Pi, if serial port needs configuration, the script will offer to fix it automatically. This requires a **reboot**, after which you run the configure script again.

### Step 3: Verify

```bash
sudo systemctl status mavlink-router
```

You should see `active (running)`. Connect your ground station to the configured UDP ports.

---

## ✅ That's It!

The configure script handles everything - platform detection, serial setup, and configuration.

---

## 📚 Documentation

| Guide | Description |
|-------|-------------|
| [UART Setup Guide](docs/UART-SETUP.md) | Detailed serial port configuration and wiring |
| [CLI Reference](docs/CLI-REFERENCE.md) | All command-line options |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | Common issues and solutions |

---

## 🔧 Advanced Usage

### Auto Mode (Minimal Prompts)

```bash
sudo ./configure_mavlink_router.sh --auto --gcs-ip 192.168.1.100
```

### Headless Mode (No Prompts)

```bash
sudo ./configure_mavlink_router.sh --headless \
    --uart /dev/ttyS0 \
    --baud 57600 \
    --endpoints "127.0.0.1:14540,127.0.0.1:14569,192.168.1.100:24550"
```

### USB Serial (No Boot Config Needed)

```bash
sudo ./configure_mavlink_router.sh --auto --uart /dev/ttyUSB0
```

### UDP Input (For Simulation)

```bash
sudo ./configure_mavlink_router.sh --headless \
    --input-type udp \
    --input-port 14550 \
    --endpoints "127.0.0.1:14540,127.0.0.1:14569"
```

### CLI Commands

```bash
./mavlink-anywhere status    # Show status
./mavlink-anywhere test      # Test serial connection
./mavlink-anywhere logs      # View logs
./mavlink-anywhere help      # All commands
```

---

## 🔌 Supported Platforms

| Platform | Serial Config | Notes |
|----------|--------------|-------|
| **Raspberry Pi** | Auto-detected | Script offers auto-fix for serial setup |
| **NVIDIA Jetson** | Manual | Ensure UART is enabled |
| **Generic Linux** | Manual | Ensure UART device exists |
| **USB Serial** | None needed | Just plug in adapter |
| **UDP Input** | None needed | For SITL/simulation |

---

## 🌐 Remote Connectivity

For internet streaming, you need:

1. **Internet** on companion computer (WiFi, 4G, Ethernet)
2. **VPN** for secure access:
   - [NetBird](https://netbird.io/) - Recommended (shown in video)
   - [Tailscale](https://tailscale.com/)
   - [WireGuard](https://www.wireguard.com/)

---

## 🤝 MDS Integration

Integrates with [MAVSDK Drone Show](https://github.com/alireza787b/mavsdk_drone_show):

```bash
sudo ./tools/mds_init.sh -d 1 -y --mavlink-auto --gcs-ip 100.96.32.75
```

---

## ❓ Need Help?

1. **[Video Tutorial](https://www.youtube.com/watch?v=_QEWpoy6HSo)** - Most common scenarios
2. **[Troubleshooting Guide](docs/TROUBLESHOOTING.md)** - Common issues
3. **[GitHub Issues](https://github.com/alireza787b/mavlink-anywhere/issues)** - Bug reports

---

## 📄 License

MIT License - Copyright (c) 2024 Alireza Ghaderi

---

<p align="center">
  <b>Made with ❤️ for the drone community</b><br>
  <a href="https://github.com/alireza787b/mavlink-anywhere">GitHub</a> •
  <a href="https://www.youtube.com/watch?v=_QEWpoy6HSo">Tutorial</a> •
  <a href="https://github.com/alireza787b/mavsdk_drone_show">MDS</a>
</p>
