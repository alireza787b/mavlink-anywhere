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

For more information, visit the [GitHub Repo](https://github.com/alireza787b/mavlink-anywhere) or contact the author at p30planets@gmail.com.
```

#### Step 4: Commit and Push the Changes to GitHub

1. **Add the Files to Git**:
- In VSCode terminal or your preferred terminal:
```sh
git add install_mavlink_router.sh configure_mavlink_router.sh README.md
```

2. **Commit the Changes**:
- In the terminal:
```sh
git commit -m "Initial commit: Add installation and configuration scripts"
```

3. **Push the Changes to GitHub**:
- In the terminal:
```sh
git push origin main
```

#### Final Verification

1. **Visit Your GitHub Repository**: Open your browser and go to `https://github.com/alireza787b/mavlink-anywhere`.

2. **Verify the Files**: Ensure that `install_mavlink_router.sh`, `configure_mavlink_router.sh`, and `README.md` are present in the repository.

### Summary

You have successfully created a GitHub repository named `mavlink-anywhere`, added installation and configuration scripts, and pushed these

changes to GitHub. The repository is now ready for use and can be shared with others.