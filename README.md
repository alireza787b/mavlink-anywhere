# MavlinkAnywhere

This project provides scripts to install and configure `mavlink-router` on a Raspberry Pi.

## Installation Script

This script installs `mavlink-router`.

### Usage

1. Clone the repository:
```sh
git clone https://github.com/alireza787b/mavlink-anywhere.git
cd mavlink-anywhere
```

2. Run the installation script:
```sh
chmod +x install_mavlink_router.sh
sudo ./install_mavlink_router.sh
```

## Configuration Script

This script generates and updates the configuration for `mavlink-router`, sets up a systemd service, and enables routing.

### Usage

1. Run the configuration script:
```sh
chmod +x configure_mavlink_router.sh
sudo ./configure_mavlink_router.sh
```

2. Follow the prompts to set up UART device, baud rate, and UDP port.

## Contact

For more information, visit the [GitHub Repo](https://github.com/alireza787b/mavlink-anywhere).
