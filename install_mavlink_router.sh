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

# Detect swap management method
SWAP_METHOD="none"
ORIGINAL_SWAP_SIZE=100
if command -v dphys-swapfile &> /dev/null && [ -f /etc/dphys-swapfile ]; then
    SWAP_METHOD="dphys"
    # Try to detect original swap size
    if grep -q "CONF_SWAPSIZE=" /etc/dphys-swapfile; then
        ORIGINAL_SWAP_SIZE=$(grep "CONF_SWAPSIZE=" /etc/dphys-swapfile | head -n1 | cut -d'=' -f2)
    fi
elif [ -f /swapfile ] || swapon --show | grep -q "/swapfile"; then
    SWAP_METHOD="standard"
else
    SWAP_METHOD="none"
fi

# Function to increase swap space
increase_swap() {
    print_progress "Increasing swap space..."

    if [ "$SWAP_METHOD" = "dphys" ]; then
        # Using dphys-swapfile (traditional Raspberry Pi method)
        sudo dphys-swapfile swapoff
        sudo sed -i 's/CONF_SWAPSIZE=[0-9]*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
        sudo dphys-swapfile setup
        sudo dphys-swapfile swapon
        echo "Swap increased to 2048MB using dphys-swapfile"
    elif [ "$SWAP_METHOD" = "standard" ]; then
        # Using standard Linux swap file
        sudo swapoff /swapfile 2>/dev/null || true
        sudo rm -f /swapfile
        sudo fallocate -l 2G /swapfile || sudo dd if=/dev/zero of=/swapfile bs=1M count=2048
        sudo chmod 600 /swapfile
        sudo mkswap /swapfile
        sudo swapon /swapfile
        echo "Swap increased to 2048MB using standard method"
    else
        # No existing swap, create new one
        echo "No existing swap detected, creating new swap file..."
        sudo fallocate -l 2G /swapfile 2>/dev/null || sudo dd if=/dev/zero of=/swapfile bs=1M count=2048
        sudo chmod 600 /swapfile
        sudo mkswap /swapfile
        sudo swapon /swapfile
        SWAP_METHOD="standard"
        echo "Created 2048MB swap file"
    fi
}

# Function to clean up swap space
cleanup_swap() {
    print_progress "Resetting swap space to original size..."

    if [ "$SWAP_METHOD" = "dphys" ]; then
        # Restore dphys-swapfile to original size
        sudo dphys-swapfile swapoff
        sudo sed -i "s/CONF_SWAPSIZE=[0-9]*/CONF_SWAPSIZE=$ORIGINAL_SWAP_SIZE/" /etc/dphys-swapfile
        sudo dphys-swapfile setup
        sudo dphys-swapfile swapon
        echo "Swap restored to ${ORIGINAL_SWAP_SIZE}MB using dphys-swapfile"
    elif [ "$SWAP_METHOD" = "standard" ]; then
        # Restore standard swap to original size (typically 100MB or 200MB)
        sudo swapoff /swapfile
        sudo rm -f /swapfile
        # Use 200MB as default for modern systems
        local restore_size=${ORIGINAL_SWAP_SIZE:-200}
        sudo fallocate -l ${restore_size}M /swapfile || sudo dd if=/dev/zero of=/swapfile bs=1M count=$restore_size
        sudo chmod 600 /swapfile
        sudo mkswap /swapfile
        sudo swapon /swapfile
        echo "Swap restored to ${restore_size}MB using standard method"
    else
        echo "No swap management needed"
    fi
}

# Stop any existing mavlink-router service
print_progress "Stopping any existing mavlink-router service..."
sudo systemctl stop mavlink-router

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
sudo apt update && sudo apt install -y git meson ninja-build pkg-config gcc g++ systemd libsystemd-dev python3-venv || { echo "Installation of packages failed"; cleanup_swap; exit 1; }

# Increase swap space for low-memory systems
increase_swap

# Clone and navigate into the repository
print_progress "Cloning mavlink-router repository..."
git clone https://github.com/mavlink-router/mavlink-router.git || { echo "Cloning of repository failed"; cleanup_swap; exit 1; }
cd mavlink-router || { echo "Changing directory failed"; cleanup_swap; exit 1; }

# Fetch dependencies (submodules)
print_progress "Fetching submodules..."
git submodule update --init --recursive || { echo "Submodule update failed"; cleanup_swap; exit 1; }

# Create and activate a virtual environment
print_progress "Creating and activating a virtual environment..."
python3 -m venv ~/mavlink-router-venv
source ~/mavlink-router-venv/bin/activate

# Install Meson in the virtual environment
print_progress "Installing Meson in the virtual environment..."
pip install meson || { echo "Meson installation failed"; cleanup_swap; deactivate; exit 1; }

# Build with Meson and Ninja
print_progress "Setting up the build with Meson..."
meson setup build . || { echo "Meson setup failed"; cleanup_swap; deactivate; exit 1; }
print_progress "Building with Ninja..."
ninja -C build || { echo "Ninja build failed"; cleanup_swap; deactivate; exit 1; }

# Install
print_progress "Installing mavlink-router..."
sudo ninja -C build install || { echo "Installation failed"; cleanup_swap; deactivate; exit 1; }

# Deactivate virtual environment and navigate back to home directory
deactivate
cd ~

# Print success message
print_progress "mavlink-router installed successfully."

# Reset swap space to original size
cleanup_swap

print_progress "Installation script completed."
echo "Next steps:"
echo "1. Configure mavlink-router using the provided configuration script."
echo "2. Check the status of the mavlink-router service with: sudo systemctl status mavlink-router"
echo "3. For detailed logs, use: sudo journalctl -u mavlink-router -f"
echo "4. Use QGroundControl to connect to the Raspberry Pi's IP address on port 14550."
