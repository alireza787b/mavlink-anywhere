# MavlinkAnywhere

This project provides scripts to install and configure `mavlink-router` on a Raspberry Pi. `mavlink-router` is an application to route MAVLink packets between different types of endpoints, such as UART, UDP, and TCP.

## Installation Script

This script installs `mavlink-router` on your Raspberry Pi, including necessary dependencies.

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

- Checks if `mavlink-router` is already installed.
- Removes any existing `mavlink-router` directory.
- Updates the system and installs required packages (git, meson, ninja-build, pkg-config, gcc, g++, systemd, python3-venv).
- Increases the swap space to ensure successful compilation on low-memory systems.
- Clones the `mavlink-router` repository and initializes its submodules.
- Creates and activates a Python virtual environment.
- Installs Meson build system in the virtual environment.
- Builds and installs `mavlink-router` using Meson and Ninja.
- Resets the swap space to its original size after installation.

## Configuration Script

This script generates and updates the configuration for `mavlink-router`, sets up a systemd service, and enables routing.

### Usage

1. **Run the configuration script:**
   ```sh
   chmod +x configure_mavlink_router.sh
   sudo ./configure_mavlink_router.sh
   ```

2. **Follow the prompts to set up UART device, baud rate, and UDP port:**
   - **UART Device**: Default is `/dev/ttyS0`. This is the default serial port on the Raspberry Pi.
   - **Baud Rate**: Default is `57600`. This is the communication speed between the Raspberry Pi and connected devices.
   - **UDP Port**: Default is `14550`. This is the port on which MAVLink packets will be sent/received over UDP.

### What the Configuration Script Does:

- Prompts the user to enable UART and disable the serial console using `raspi-config`.
- Prompts for UART device, baud rate, and UDP port.
- Creates an environment file with the provided values.
- Generates the `mavlink-router` configuration file.
- Creates an interactive script for future updates if needed.
- Stops any existing `mavlink-router` service.
- Creates a systemd service file to manage the `mavlink-router` service.
- Reloads systemd, enables, and starts the `mavlink-router` service.

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

Use QGroundControl to connect to the Raspberry Pi's IP address on the specified UDP port (default: `14550`).

## Contact

For more information, visit the [GitHub Repo](https://github.com/alireza787b/mavlink-anywhere).
```
