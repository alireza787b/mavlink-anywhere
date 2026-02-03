# MAVLink Anywhere

**Stream MAVLink telemetry from your drone to anywhere in the world.**

Route MAVLink data from your flight controller (Pixhawk/ArduPilot/PX4) through a companion computer (Raspberry Pi, Jetson, etc.) to ground stations, SDKs, and remote locations over the internet.

---

## 📺 Video Tutorial

**First time here? Watch the video - it covers everything!**

[![MAVLink Anywhere Tutorial](https://img.youtube.com/vi/_QEWpoy6HSo/0.jpg)](https://www.youtube.com/watch?v=_QEWpoy6HSo)

🎬 [Complete Setup Guide (YouTube)](https://www.youtube.com/watch?v=_QEWpoy6HSo)

---

## 🚀 Quick Start

Follow these steps on your Raspberry Pi (or companion computer):

### Step 1: Clone the Repository

```bash
git clone https://github.com/alireza787b/mavlink-anywhere.git
cd mavlink-anywhere
```

### Step 2: Install mavlink-router

```bash
sudo ./install_mavlink_router.sh
```

This builds and installs the mavlink-router binary. Takes 5-10 minutes on Raspberry Pi.

### Step 3: Configure Serial Port (Raspberry Pi)

Before configuring mavlink-router, ensure your serial port is ready:

```bash
sudo raspi-config
```

Navigate to:
- **Interface Options** → **Serial Port**
- "Login shell accessible over serial?" → **No**
- "Serial port hardware enabled?" → **Yes**

Then **reboot**:
```bash
sudo reboot
```

> 📖 **Need help?** See [UART Setup Guide](docs/UART-SETUP.md) for detailed instructions and wiring diagrams.

### Step 4: Configure mavlink-router

After reboot, run the configuration script:

```bash
cd mavlink-anywhere
sudo ./configure_mavlink_router.sh
```

The interactive prompts will guide you through:
1. Selecting your UART device (e.g., `/dev/ttyS0`)
2. Setting baud rate (typically `57600`)
3. Adding UDP endpoints for your ground station

### Step 5: Verify It's Working

```bash
sudo systemctl status mavlink-router
```

You should see `active (running)`. Connect QGroundControl to your Pi's IP address on port `14550`.

---

## ✅ That's It!

Your MAVLink data is now being routed. Connect your ground station software to the configured UDP endpoints.

**Need remote/internet access?** See the [video tutorial](https://www.youtube.com/watch?v=_QEWpoy6HSo) for VPN setup with [NetBird](https://netbird.io/).

---

## 📚 Documentation

| Guide | Description |
|-------|-------------|
| [UART Setup Guide](docs/UART-SETUP.md) | Serial port configuration, wiring, Raspberry Pi setup |
| [CLI Reference](docs/CLI-REFERENCE.md) | Advanced command-line options and automation |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | Common issues and solutions |

---

## 🔧 Advanced Usage

### Auto Mode (Skip Prompts)

For quick setup with automatic UART detection:

```bash
sudo ./configure_mavlink_router.sh --auto --gcs-ip YOUR_GCS_IP
```

### Headless Mode (Scripted Automation)

For automated deployments with no prompts:

```bash
sudo ./configure_mavlink_router.sh --headless \
    --uart /dev/ttyS0 \
    --baud 57600 \
    --endpoints "127.0.0.1:14540,127.0.0.1:14569,192.168.1.100:24550"
```

### UDP Input (For Simulation/SITL)

Use UDP input instead of serial:

```bash
sudo ./configure_mavlink_router.sh --headless \
    --input-type udp \
    --input-port 14550 \
    --endpoints "127.0.0.1:14540,127.0.0.1:14569"
```

### CLI Commands

```bash
./mavlink-anywhere status    # Show current status
./mavlink-anywhere test      # Test serial connection
./mavlink-anywhere logs      # View service logs
./mavlink-anywhere help      # See all commands
```

> 📖 See [CLI Reference](docs/CLI-REFERENCE.md) for complete documentation.

---

## 🔌 Hardware Requirements

- **Companion Computer**: Raspberry Pi (any model), Jetson, or similar
- **Flight Controller**: Pixhawk, ArduPilot, PX4, or MAVLink-compatible
- **Connection**: UART cable from flight controller TELEM port to companion computer

---

## 🌐 Remote Connectivity Options

To stream telemetry over the internet, you need:

1. **Internet connection** on your companion computer:
   - WiFi (use [Smart WiFi Manager](https://github.com/alireza787b/smart-wifi-manager) for reliability)
   - 4G/LTE USB dongle
   - Ethernet

2. **VPN** for secure remote access (recommended):
   - [NetBird](https://netbird.io/) - Easiest, shown in video tutorial
   - [Tailscale](https://tailscale.com/)
   - [WireGuard](https://www.wireguard.com/)
   - [ZeroTier](https://www.zerotier.com/)

---

## 🤝 Integration with MDS

mavlink-anywhere integrates with [MAVSDK Drone Show (MDS)](https://github.com/alireza787b/mavsdk_drone_show) for automated fleet setup:

```bash
sudo ./tools/mds_init.sh -d 1 -y --mavlink-auto --gcs-ip 100.96.32.75
```

---

## ❓ Need Help?

1. **Watch the [video tutorial](https://www.youtube.com/watch?v=_QEWpoy6HSo)** - covers most setup scenarios
2. **Check [Troubleshooting Guide](docs/TROUBLESHOOTING.md)** - common issues and solutions
3. **Open a [GitHub Issue](https://github.com/alireza787b/mavlink-anywhere/issues)** - for bugs or questions

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
