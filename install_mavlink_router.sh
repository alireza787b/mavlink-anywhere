#!/bin/bash

echo "================================================================="
echo "MavlinkAnywhere: Mavlink-router Installation Script"
echo "Author: Alireza Ghaderi"
echo "GitHub: https://github.com/alireza787b/mavlink-anywhere"
echo "Contact: p30planets@gmail.com"
echo "================================================================="

# Function to print progress messages
print_progress() {
    echo "================================================================="
    echo "$1"
    echo "================================================================="
}

# Navigate to home directory
cd ~

# Check if mavlink-router is already installed
if command -v mavlink-routerd &> /dev/null; then
    print_progress "mavlink-router is already installed. You're good to go!"
    exit 0
fi

# If the mavlink-router directory exists, remove it
if [ -d "mavlink-router" ]; then
    print_progress "Removing existing mavlink-router directory..."
    rm -rf mavlink-router
fi

# Update and install packages
print_progress "Updating and installing necessary packages..."
sudo apt update && sudo apt install -y git meson ninja-build pkg-config gcc g++ systemd python3-venv || { echo "Installation of packages failed"; exit 1; }

# Increase swap space for low-memory systems
print_progress "Increasing swap space..."
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=[0-9]*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup
sudo dphys-swapfile swapon

# Clone and navigate into the repository
print_progress "Cloning mavlink-router repository..."
git clone https://github.com/mavlink-router/mavlink-router.git || { echo "Cloning of repository failed"; exit 1; }
cd mavlink-router || { echo "Changing directory failed"; exit 1; }

# Fetch dependencies (submodules)
print_progress "Fetching submodules..."
git submodule update --init --recursive || { echo "Submodule update failed"; exit 1; }

# Create and activate a virtual environment
print_progress "Creating and activating a virtual environment..."
python3 -m venv ~/mavlink-router-venv
source ~/mavlink-router-venv/bin/activate

# Install Meson in the virtual environment
print_progress "Installing Meson in the virtual environment..."
pip install meson || { echo "Meson installation failed"; exit 1; }

# Build with Meson and Ninja
print_progress "Setting up the build with Meson..."
meson setup build . || { echo "Meson setup failed"; exit 1; }
print_progress "Building with Ninja..."
ninja -C build || { echo "Ninja build failed"; exit 1; }

# Install
print_progress "Installing mavlink-router..."
sudo ninja -C build install || { echo "Installation failed"; exit 1; }

# Deactivate virtual environment and navigate back to home directory
deactivate
cd ~

# Print success message
print_progress "mavlink-router installed successfully."

# Reset swap space to original size
print_progress "Resetting swap space to original size..."
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=2048/CONF_SWAPSIZE=100/' /etc/dphys-swapfile  # Assuming original size is 100
sudo dphys-swapfile setup
sudo dphys-swapfile swapon

print_progress "Installation script completed."
