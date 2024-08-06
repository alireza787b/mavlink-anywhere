#!/bin/bash

echo "================================================================="
echo "MavlinkAnywhere: Mavlink-router Configuration Script"
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

# Remind user to enable UART and disable serial console
print_progress "If you are going to use ttyS0, ensure that UART is enabled and the serial console is disabled."
echo "You can enable UART and disable the serial console using raspi-config."
echo "1. Run: sudo raspi-config"
echo "2. Navigate to: Interface Options -> Serial Port"
echo "3. Disable the serial console and enable the serial port hardware"
echo "4. Reboot the Raspberry Pi after making these changes."
read -p "Press Enter if you are ready to continue..."

# Check if an existing configuration file is available and read it
CONFIG_FILE="/etc/mavlink-router/main.conf"
if [ -f "$CONFIG_FILE" ]; then
    print_progress "Existing configuration file found. Reading current settings..."
    source /etc/default/mavlink-router
    DEFAULT_UART_DEVICE=${UART_DEVICE}
    DEFAULT_UART_BAUD=${UART_BAUD}
    DEFAULT_UDP_ENDPOINTS=${UDP_ENDPOINTS}
else
    DEFAULT_UART_DEVICE="/dev/ttyS0"
    DEFAULT_UART_BAUD="57600"
    DEFAULT_UDP_ENDPOINTS="0.0.0.0:14550"
fi

# Step 1: Prompt for UART device, baud rate, and UDP endpoints using existing settings as defaults
read -p "Enter UART device (default: ${DEFAULT_UART_DEVICE}): " UART_DEVICE
UART_DEVICE=${UART_DEVICE:-$DEFAULT_UART_DEVICE}

read -p "Enter UART baud rate (default: ${DEFAULT_UART_BAUD}): " UART_BAUD
UART_BAUD=${UART_BAUD:-$DEFAULT_UART_BAUD}

read -p "Enter UDP endpoints (default: ${DEFAULT_UDP_ENDPOINTS}). You can enter multiple endpoints separated by spaces: " UDP_ENDPOINTS
UDP_ENDPOINTS=${UDP_ENDPOINTS:-$DEFAULT_UDP_ENDPOINTS}

# Step 2: Create the environment file
print_progress "Creating environment file with the provided values..."
sudo mkdir -p /etc/default
sudo bash -c "cat <<EOF > /etc/default/mavlink-router
UART_DEVICE=${UART_DEVICE}
UART_BAUD=${UART_BAUD}
UDP_ENDPOINTS=\"${UDP_ENDPOINTS}\"
EOF"

# Step 3: Create the configuration file directly
print_progress "Creating configuration file..."
sudo mkdir -p /etc/mavlink-router
sudo bash -c "cat <<EOF > /etc/mavlink-router/main.conf
[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=${UART_DEVICE}
Baud=${UART_BAUD}
EOF"

# Add UDP endpoints to the configuration file
IFS=' ' read -r -a ENDPOINT_ARRAY <<< "${UDP_ENDPOINTS}"
for ENDPOINT in "${ENDPOINT_ARRAY[@]}"; do
    sudo bash -c "cat <<EOF >> /etc/mavlink-router/main.conf
[UdpEndpoint udp]
Mode=normal
Address=$(echo ${ENDPOINT} | cut -d':' -f1)
Port=$(echo ${ENDPOINT} | cut -d':' -f2)
EOF"
done

# Step 4: Create the interactive script (for future updates if needed)
print_progress "Creating interactive script..."
sudo bash -c "cat <<EOF > /usr/bin/generate_mavlink_config.sh
#!/bin/bash

# Load existing environment variables
source /etc/default/mavlink-router

# Generate configuration from template
envsubst < /etc/mavlink-router/main.conf.template > /etc/mavlink-router/main.conf

# Verify configuration file is correctly populated
if ! grep -q '\\\$' /etc/mavlink-router/main.conf; then
    echo "Configuration file generated successfully."
else
    echo "Error: Configuration file contains unresolved variables."
    exit 1
fi
EOF"

# Make the script executable
sudo chmod +x /usr/bin/generate_mavlink_config.sh

# Step 5: Stop the service if it's already running
print_progress "Stopping any existing mavlink-router service..."
sudo systemctl stop mavlink-router

# Step 6: Create the systemd service file
print_progress "Creating systemd service file..."
sudo bash -c "cat <<EOF > /etc/systemd/system/mavlink-router.service
[Unit]
Description=MAVLink Router Service
After=network.target

[Service]
EnvironmentFile=/etc/default/mavlink-router
ExecStart=/usr/bin/mavlink-routerd -c /etc/mavlink-router/main.conf
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF"

# Step 7: Reload systemd, enable, and start the service
print_progress "Reloading systemd, enabling, and starting mavlink-router service..."
sudo systemctl daemon-reload
sudo systemctl enable mavlink-router
sudo systemctl start mavlink-router

# Print success message
print_progress "mavlink-router service installed and started successfully."
echo "You can check the status with: sudo systemctl status mavlink-router"
echo "Use QGroundControl to connect to the Raspberry Pi's IP address on the configured UDP endpoints."
echo "For more detailed logs, you can use: sudo journalctl -u mavlink-router -f"
echo "Configuration file is located at: /etc/mavlink-router/main.conf"
echo "You can manually edit the configuration file if needed."
echo "Final configuration file content:"
cat /etc/mavlink-router/main.conf
