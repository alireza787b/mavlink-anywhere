#!/bin/bash

echo "================================================================="
echo "MavlinkAnywhere: Mavlink-router Configuration Script"
echo "Author: Alireza Ghaderi"
echo "GitHub: https://github.com/alireza787b/mavlink-anywhere"
echo "Contact: p30planets@gmail.com"
echo "================================================================="

# Step 1: Prompt for UART device, baud rate, and UDP port
read -p "Enter UART device (default: /dev/serial0): " UART_DEVICE
UART_DEVICE=${UART_DEVICE:-/dev/serial0}

read -p "Enter UART baud rate (default: 57600): " UART_BAUD
UART_BAUD=${UART_BAUD:-57600}

read -p "Enter UDP port (default: 14550): " UDP_PORT
UDP_PORT=${UDP_PORT:-14550}

# Step 2: Create the environment file
echo "Creating environment file with the provided values..."
sudo bash -c "cat <<EOF > /etc/default/mavlink-router
UART_DEVICE=${UART_DEVICE}
UART_BAUD=${UART_BAUD}
UDP_PORT=${UDP_PORT}
EOF"

# Step 3: Create the configuration template
echo "Creating configuration template..."
sudo mkdir -p /etc/mavlink-router
sudo bash -c "cat <<EOF > /etc/mavlink-router/main.conf.template
[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=\${UART_DEVICE}
Baud=\${UART_BAUD}

[UdpEndpoint udp]
Address=0.0.0.0
Port=\${UDP_PORT}
EOF"

# Step 4: Create the interactive script
echo "Creating interactive script..."
sudo bash -c "cat <<EOF > /usr/local/bin/generate_mavlink_config.sh
#!/bin/bash

# Load existing environment variables
source /etc/default/mavlink-router

# Generate configuration from template
envsubst < /etc/mavlink-router/main.conf.template > /etc/mavlink-router/main.conf
EOF"

# Make the script executable
sudo chmod +x /usr/local/bin/generate_mavlink_config.sh

# Step 5: Create the systemd service file
echo "Creating systemd service file..."
sudo bash -c "cat <<EOF > /etc/systemd/system/mavlink-router.service
[Unit]
Description=MAVLink Router Service
After=network.target

[Service]
EnvironmentFile=/etc/default/mavlink-router
ExecStartPre=/usr/local/bin/generate_mavlink_config.sh
ExecStart=/usr/local/bin/mavlink-routerd -c /etc/mavlink-router/main.conf
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF"

# Reload systemd, enable, and start the service
echo "Reloading systemd, enabling, and starting mavlink-router service..."
sudo systemctl daemon-reload
sudo systemctl enable mavlink-router
sudo systemctl start mavlink-router

# Print success message
echo "mavlink-router service installed and started successfully."
echo "You can check the status with: sudo systemctl status mavlink-router"
echo "Use QGroundControl to connect to the Raspberry Pi's IP address on port ${UDP_PORT}."
