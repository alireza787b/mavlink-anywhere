#!/bin/bash
# =============================================================================
# MAVLink-Anywhere: Mavlink-router Configuration Script
# =============================================================================
# Version: 2.0.0
# Author: Alireza Ghaderi
# GitHub: https://github.com/alireza787b/mavlink-anywhere
# Contact: p30planets@gmail.com
# =============================================================================
#
# This script configures mavlink-router with support for three modes:
#
# 1. Interactive Mode (default - UNCHANGED from v1.x):
#    sudo ./configure_mavlink_router.sh
#    - Prompts for all settings
#    - Same behavior as original script (YouTube tutorial compatible)
#
# 2. Headless Mode (NEW):
#    sudo ./configure_mavlink_router.sh --headless \
#        --uart /dev/ttyS0 \
#        --baud 57600 \
#        --endpoints "127.0.0.1:14540,127.0.0.1:14569,100.96.32.75:24550"
#    - No prompts, all settings via command line
#    - For scripted/automated deployments
#
# 3. Auto Mode (NEW):
#    sudo ./configure_mavlink_router.sh --auto --gcs-ip 100.96.32.75
#    - Auto-detects UART device
#    - Uses standard MDS endpoints
#    - Only requires GCS IP (optional)
#
# 4. UDP Input Mode (NEW):
#    sudo ./configure_mavlink_router.sh --headless \
#        --input-type udp \
#        --input-port 14550 \
#        --endpoints "127.0.0.1:14540,127.0.0.1:14569"
#    - For SITL or network-bridged setups
#
# =============================================================================

set -e

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# =============================================================================
# SOURCE LIBRARIES (if available)
# =============================================================================

LIB_DIR="${SCRIPT_DIR}/lib"
if [[ -d "$LIB_DIR" ]]; then
    source "${LIB_DIR}/common.sh" 2>/dev/null || true
    source "${LIB_DIR}/detect.sh" 2>/dev/null || true
    source "${LIB_DIR}/config.sh" 2>/dev/null || true
    source "${LIB_DIR}/service.sh" 2>/dev/null || true
    LIBS_LOADED=true
else
    LIBS_LOADED=false
fi

# =============================================================================
# FALLBACK FUNCTIONS (when libraries not available)
# =============================================================================

if [[ "$LIBS_LOADED" != "true" ]]; then
    # Minimal print function
    print_progress() {
        echo "================================================================="
        echo "$1"
        echo "================================================================="
    }
fi

# =============================================================================
# PARSE COMMAND LINE ARGUMENTS
# =============================================================================

HEADLESS=false
AUTO_MODE=false
INPUT_TYPE="uart"
UART_DEVICE=""
UART_BAUD=""
ENDPOINTS=""
GCS_IP=""
INPUT_ADDRESS="0.0.0.0"
INPUT_PORT="14550"
SHOW_HELP=false

show_help() {
    cat <<EOF
MAVLink-Anywhere Configuration Script v2.0.0

Usage: sudo ./configure_mavlink_router.sh [OPTIONS]

Modes:
  (default)         Interactive mode - prompts for all settings
  --headless        Headless mode - no prompts, all settings via CLI
  --auto            Auto mode - auto-detect UART, use standard endpoints

Options:
  --uart DEVICE     UART device path (e.g., /dev/ttyS0, /dev/ttyAMA0)
  --baud RATE       Baud rate (default: 57600)
  --endpoints LIST  Comma-separated endpoints (e.g., "127.0.0.1:14540,192.168.1.100:24550")
  --gcs-ip IP       GCS IP address (adds endpoint on port 24550)

UDP Input Options:
  --input-type TYPE Input type: uart (default) or udp
  --input-address   UDP listen address (default: 0.0.0.0)
  --input-port      UDP listen port (default: 14550)

Other:
  -h, --help        Show this help message
  --debug           Enable debug output

Examples:
  # Interactive (original behavior)
  sudo ./configure_mavlink_router.sh

  # Headless with UART
  sudo ./configure_mavlink_router.sh --headless \\
      --uart /dev/ttyS0 --baud 57600 \\
      --endpoints "127.0.0.1:14540,127.0.0.1:14569,100.96.32.75:24550"

  # Auto-detect with GCS IP
  sudo ./configure_mavlink_router.sh --auto --gcs-ip 100.96.32.75

  # UDP input for SITL
  sudo ./configure_mavlink_router.sh --headless \\
      --input-type udp --input-port 14550 \\
      --endpoints "127.0.0.1:14540,127.0.0.1:14569"

Documentation: https://github.com/alireza787b/mavlink-anywhere
EOF
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --headless)
            HEADLESS=true
            shift
            ;;
        --auto)
            AUTO_MODE=true
            shift
            ;;
        --uart)
            UART_DEVICE="$2"
            shift 2
            ;;
        --baud)
            UART_BAUD="$2"
            shift 2
            ;;
        --endpoints)
            ENDPOINTS="$2"
            shift 2
            ;;
        --gcs-ip)
            GCS_IP="$2"
            shift 2
            ;;
        --input-type)
            INPUT_TYPE="$2"
            shift 2
            ;;
        --input-address)
            INPUT_ADDRESS="$2"
            shift 2
            ;;
        --input-port)
            INPUT_PORT="$2"
            shift 2
            ;;
        --debug)
            MA_DEBUG=true
            shift
            ;;
        -h|--help)
            SHOW_HELP=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

if [[ "$SHOW_HELP" == "true" ]]; then
    show_help
    exit 0
fi

# =============================================================================
# MODE ROUTING
# =============================================================================

if [[ "$HEADLESS" == "true" ]]; then
    # =========================================================================
    # HEADLESS MODE - No prompts, all settings from CLI
    # =========================================================================

    echo "================================================================="
    echo "MavlinkAnywhere: Mavlink-router Configuration (Headless Mode)"
    echo "Author: Alireza Ghaderi"
    echo "GitHub: https://github.com/alireza787b/mavlink-anywhere"
    echo "================================================================="

    # Validate required parameters based on input type
    if [[ "$INPUT_TYPE" == "uart" ]]; then
        if [[ -z "$UART_DEVICE" ]]; then
            echo "Error: --uart is required in headless mode for UART input"
            exit 1
        fi
        if [[ -z "$UART_BAUD" ]]; then
            UART_BAUD="57600"
        fi
    elif [[ "$INPUT_TYPE" == "udp" ]]; then
        if [[ -z "$INPUT_PORT" ]]; then
            INPUT_PORT="14550"
        fi
    else
        echo "Error: Invalid input type: $INPUT_TYPE (use 'uart' or 'udp')"
        exit 1
    fi

    if [[ -z "$ENDPOINTS" ]]; then
        echo "Error: --endpoints is required in headless mode"
        exit 1
    fi

    # Use library functions if available, otherwise inline
    if [[ "$LIBS_LOADED" == "true" ]]; then
        if [[ "$INPUT_TYPE" == "uart" ]]; then
            configure_uart_headless "$UART_DEVICE" "$UART_BAUD" "$ENDPOINTS"
            setup_service
        else
            configure_udp_headless "$INPUT_ADDRESS" "$INPUT_PORT" "$ENDPOINTS"
            setup_service
        fi
    else
        # Inline headless configuration (fallback)
        print_progress "Creating configuration (headless)..."

        sudo mkdir -p /etc/mavlink-router
        sudo mkdir -p /etc/default

        if [[ "$INPUT_TYPE" == "uart" ]]; then
            # Generate UART config
            sudo bash -c "cat > /etc/mavlink-router/main.conf" <<EOF
[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=${UART_DEVICE}
Baud=${UART_BAUD}
EOF
        else
            # Generate UDP input config
            sudo bash -c "cat > /etc/mavlink-router/main.conf" <<EOF
[General]
TcpServerPort=5760
ReportStats=false

[UdpEndpoint input]
Mode=server
Address=${INPUT_ADDRESS}
Port=${INPUT_PORT}
EOF
        fi

        # Add UDP endpoints
        IFS=',' read -r -a ENDPOINT_ARRAY <<< "${ENDPOINTS}"
        INDEX=1
        for ENDPOINT in "${ENDPOINT_ARRAY[@]}"; do
            ENDPOINT=$(echo "$ENDPOINT" | tr -d ' ')
            ADDRESS=$(echo ${ENDPOINT} | cut -d':' -f1)
            PORT=$(echo ${ENDPOINT} | cut -d':' -f2)
            sudo bash -c "cat >> /etc/mavlink-router/main.conf" <<EOF

[UdpEndpoint udp${INDEX}]
Mode=normal
Address=${ADDRESS}
Port=${PORT}
EOF
            INDEX=$((INDEX+1))
        done

        # Write env file
        sudo bash -c "cat > /etc/default/mavlink-router" <<EOF
UART_DEVICE=${UART_DEVICE}
UART_BAUD=${UART_BAUD}
UDP_ENDPOINTS="${ENDPOINTS}"
INPUT_TYPE=${INPUT_TYPE}
INPUT_ADDRESS=${INPUT_ADDRESS}
INPUT_PORT=${INPUT_PORT}
EOF

        # Setup service
        print_progress "Setting up systemd service..."
        sudo bash -c "cat > /etc/systemd/system/mavlink-router.service" <<EOF
[Unit]
Description=MAVLink Router Service
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/default/mavlink-router
ExecStart=/usr/bin/mavlink-routerd -c /etc/mavlink-router/main.conf
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

        sudo systemctl daemon-reload
        sudo systemctl enable mavlink-router
        sudo systemctl restart mavlink-router
    fi

    echo ""
    print_progress "Configuration complete (headless mode)"
    echo "Configuration file: /etc/mavlink-router/main.conf"
    echo ""
    cat /etc/mavlink-router/main.conf

elif [[ "$AUTO_MODE" == "true" ]]; then
    # =========================================================================
    # AUTO MODE - Auto-detect settings, minimal prompts
    # =========================================================================

    echo "================================================================="
    echo "MavlinkAnywhere: Mavlink-router Configuration (Auto Mode)"
    echo "Author: Alireza Ghaderi"
    echo "GitHub: https://github.com/alireza787b/mavlink-anywhere"
    echo "================================================================="

    # Auto-detect UART device
    if [[ "$LIBS_LOADED" == "true" ]]; then
        UART_DEVICE=$(detect_uart_device)
        echo "Auto-detected UART device: $UART_DEVICE"
    else
        # Fallback detection
        if [[ -e /dev/serial0 ]]; then
            UART_DEVICE=$(readlink -f /dev/serial0)
        elif [[ -e /dev/ttyS0 ]]; then
            UART_DEVICE="/dev/ttyS0"
        elif [[ -e /dev/ttyAMA0 ]]; then
            UART_DEVICE="/dev/ttyAMA0"
        else
            UART_DEVICE="/dev/ttyS0"
        fi
        echo "Auto-detected UART device: $UART_DEVICE"
    fi

    # Use default baud rate
    UART_BAUD="57600"
    echo "Using baud rate: $UART_BAUD"

    # Build endpoints list (standard MDS endpoints)
    ENDPOINTS="127.0.0.1:14540,127.0.0.1:14569,127.0.0.1:12550"

    # Add GCS endpoint if provided
    if [[ -n "$GCS_IP" ]]; then
        ENDPOINTS="${ENDPOINTS},${GCS_IP}:24550"
        echo "GCS endpoint: ${GCS_IP}:24550"
    else
        # Prompt for GCS IP if not provided
        echo ""
        read -p "Enter GCS IP address (or press Enter to skip): " GCS_IP
        if [[ -n "$GCS_IP" ]]; then
            ENDPOINTS="${ENDPOINTS},${GCS_IP}:24550"
        fi
    fi

    echo ""
    echo "Endpoints: $ENDPOINTS"
    echo ""

    # Now configure using the detected/default values
    if [[ "$LIBS_LOADED" == "true" ]]; then
        configure_uart_headless "$UART_DEVICE" "$UART_BAUD" "$ENDPOINTS"
        setup_service
    else
        # Fallback to inline configuration
        sudo mkdir -p /etc/mavlink-router
        sudo mkdir -p /etc/default

        sudo bash -c "cat > /etc/mavlink-router/main.conf" <<EOF
[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=${UART_DEVICE}
Baud=${UART_BAUD}
EOF

        IFS=',' read -r -a ENDPOINT_ARRAY <<< "${ENDPOINTS}"
        INDEX=1
        for ENDPOINT in "${ENDPOINT_ARRAY[@]}"; do
            ENDPOINT=$(echo "$ENDPOINT" | tr -d ' ')
            ADDRESS=$(echo ${ENDPOINT} | cut -d':' -f1)
            PORT=$(echo ${ENDPOINT} | cut -d':' -f2)
            sudo bash -c "cat >> /etc/mavlink-router/main.conf" <<EOF

[UdpEndpoint udp${INDEX}]
Mode=normal
Address=${ADDRESS}
Port=${PORT}
EOF
            INDEX=$((INDEX+1))
        done

        sudo bash -c "cat > /etc/default/mavlink-router" <<EOF
UART_DEVICE=${UART_DEVICE}
UART_BAUD=${UART_BAUD}
UDP_ENDPOINTS="${ENDPOINTS}"
EOF

        sudo bash -c "cat > /etc/systemd/system/mavlink-router.service" <<EOF
[Unit]
Description=MAVLink Router Service
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/default/mavlink-router
ExecStart=/usr/bin/mavlink-routerd -c /etc/mavlink-router/main.conf
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

        sudo systemctl daemon-reload
        sudo systemctl enable mavlink-router
        sudo systemctl restart mavlink-router
    fi

    print_progress "Configuration complete (auto mode)"
    echo "Configuration file: /etc/mavlink-router/main.conf"
    echo ""
    cat /etc/mavlink-router/main.conf

else
    # =========================================================================
    # INTERACTIVE MODE - Original behavior (UNCHANGED for backward compatibility)
    # =========================================================================

    echo "================================================================="
    echo "MavlinkAnywhere: Mavlink-router Configuration Script"
    echo "Author: Alireza Ghaderi"
    echo "GitHub: https://github.com/alireza787b/mavlink-anywhere"
    echo "Contact: p30planets@gmail.com"
    echo "================================================================="

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
        source /etc/default/mavlink-router 2>/dev/null || true
        DEFAULT_UART_DEVICE=${UART_DEVICE:-/dev/ttyS0}
        DEFAULT_UART_BAUD=${UART_BAUD:-57600}
        DEFAULT_UDP_ENDPOINTS=${UDP_ENDPOINTS:-0.0.0.0:14550}
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
    INDEX=1
    for ENDPOINT in "${ENDPOINT_ARRAY[@]}"; do
        sudo bash -c "cat <<EOF >> /etc/mavlink-router/main.conf
[UdpEndpoint udp${INDEX}]
Mode=normal
Address=$(echo ${ENDPOINT} | cut -d':' -f1)
Port=$(echo ${ENDPOINT} | cut -d':' -f2)
EOF"
        INDEX=$((INDEX+1))
    done

    # Step 4: Create the interactive script (for future updates if needed)
    print_progress "Creating interactive script..."
    sudo bash -c "cat <<'EOF' > /usr/bin/generate_mavlink_config.sh
#!/bin/bash

# Load existing environment variables
source /etc/default/mavlink-router

# Generate configuration from template
envsubst < /etc/mavlink-router/main.conf.template > /etc/mavlink-router/main.conf

# Verify configuration file is correctly populated
if ! grep -q '\\\$' /etc/mavlink-router/main.conf; then
    echo \"Configuration file generated successfully.\"
else
    echo \"Error: Configuration file contains unresolved variables.\"
    exit 1
fi
EOF"

    # Make the script executable
    sudo chmod +x /usr/bin/generate_mavlink_config.sh

    # Step 5: Stop the service if it's already running
    print_progress "Stopping any existing mavlink-router service..."
    sudo systemctl stop mavlink-router 2>/dev/null || true

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
fi
