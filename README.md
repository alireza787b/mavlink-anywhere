# MAVLinkAnywhere

MAVLinkAnywhere is a general-purpose project that enables MAVLink data streaming to both local endpoints and remote locations over the internet. This project provides simplified scripts to install and configure `mavlink-router` on companion computers (Raspberry Pi, Jetson, etc.). `mavlink-router` is a powerful application that routes MAVLink packets between various endpoints, including UART, UDP, and TCP, making it ideal for MAVLink based UAV (PX4, Ardupilot, etc.) and drone communication systems.

[![MAVLinkAnywhere Tutorial](https://img.youtube.com/vi/_QEWpoy6HSo/0.jpg)](https://www.youtube.com/watch?v=_QEWpoy6HSo)

## Video Tutorial and Setup Guide
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

## Installation Script
Our installation script seamlessly installs `mavlink-router` on your companion computer, taking care of all necessary dependencies and configurations.

### Usage
1. **Clone the repository:**
   ```sh
   git clone https://github.com/alireza787b/mavlink-anywhere.git
   cd mavlink-anywhere
   ```
2. **Run the installation script:**
   ```sh
   chmod +x install_mavlink_router.sh
   sudo ./install_mavlink_router.sh
   ```

### What the Installation Script Does:
- Checks if `mavlink-router` is already installed
- Removes any existing `mavlink-router` directory
- Updates the system and installs required packages (`git`, `meson`, `ninja-build`, `pkg-config`, `gcc`, `g++`, `systemd`, `libsystemd-dev`, `python3-venv`)
- Intelligently manages swap space during compilation (supports both legacy `dphys-swapfile` and modern swap methods)
- Clones the `mavlink-router` repository and initializes its submodules
- Creates and activates a Python virtual environment
- Installs the Meson build system in the virtual environment
- Builds and installs `mavlink-router` using Meson and Ninja
- Resets the swap space to its original size after installation

**Note:** The installation script is compatible with all Debian/Raspbian versions, including the latest Bookworm release (2024+). It automatically detects your system's swap management method and adjusts accordingly.

## Configuration Script
The configuration script generates and updates the `mavlink-router` configuration, sets up a systemd service, and enables routing with flexible endpoint settings.

### Usage
1. **Run the configuration script:**
   ```sh
   chmod +x configure_mavlink_router.sh
   sudo ./configure_mavlink_router.sh
   ```
2. **Follow the prompts to set up UART device, baud rate, and UDP endpoints:**
   - If an existing configuration is found, the script will use these values as defaults and show them to you
   - **UART Device**: Default is `/dev/ttyS0`. This is the default serial port on the Raspberry Pi
   - **Baud Rate**: Default is `57600`. This is the communication speed between the companion computer and connected devices
   - **UDP Endpoints**: Default is `0.0.0.0:14550`. You can enter multiple endpoints separated by spaces (e.g., `100.110.200.3:14550 100.110.220.4:14550`)

### What the Configuration Script Does:
- Prompts the user to enable UART and disable the serial console using `raspi-config`
- Reads existing configuration values if available, and uses them as defaults
- Prompts for UART device, baud rate, and UDP endpoints
- Creates an environment file with the provided values
- Generates the `mavlink-router` configuration file
- Creates an interactive script for future updates if needed
- Stops any existing `mavlink-router` service
- Creates a systemd service file to manage the `mavlink-router` service
- Reloads systemd, enables, and starts the `mavlink-router` service

### Monitoring and Logs
- **Check the status of the service:**
  ```sh
  sudo systemctl status mavlink-router
  ```
- **View detailed logs:**
  ```sh
  sudo journalctl -u mavlink-router -f
  ```

### Connecting with QGroundControl
Use QGroundControl to connect to your companion computer's IP address on the configured UDP endpoints. For internet-based telemetry, make sure to follow the setup video to properly register your devices on your chosen VPN system or configure port forwarding on your router.

## Contact
For more information, visit the [GitHub Repo](https://github.com/alireza787b/mavlink-anywhere).

## Related Resources
- [Smart WiFi Manager Project](https://github.com/alireza787b/smart-wifi-manager)
- [NetBird VPN](https://netbird.io/)
- [Original 2020 Tutorial (Legacy Method)](https://www.youtube.com/watch?v=WoRce4Re3Wg)

## Support
If you encounter any issues, please:
1. Check the video tutorial timestamps for specific setup steps
2. Review the relevant sections in this documentation
3. Open an issue on GitHub with detailed information about your setup

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
