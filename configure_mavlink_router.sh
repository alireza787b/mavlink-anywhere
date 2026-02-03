#!/bin/bash
# =============================================================================
# MAVLink-Anywhere: Mavlink-router Configuration Script
# =============================================================================
# Version: 2.0.1
# Author: Alireza Ghaderi
# GitHub: https://github.com/alireza787b/mavlink-anywhere
# =============================================================================
#
# This script configures mavlink-router with support for multiple modes:
#
# 1. Interactive Mode (default):
#    sudo ./configure_mavlink_router.sh
#    - Checks serial prerequisites (with auto-fix option on Raspberry Pi)
#    - Prompts for settings
#
# 2. Headless Mode:
#    sudo ./configure_mavlink_router.sh --headless \
#        --uart /dev/ttyS0 --baud 57600 \
#        --endpoints "127.0.0.1:14540,192.168.1.100:24550"
#
# 3. Auto Mode:
#    sudo ./configure_mavlink_router.sh --auto --gcs-ip 100.96.32.75
#
# 4. UDP Input Mode (no serial required):
#    sudo ./configure_mavlink_router.sh --headless \
#        --input-type udp --input-port 14550 \
#        --endpoints "127.0.0.1:14540"
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
# UTILITY FUNCTIONS (always defined)
# =============================================================================

# Print progress message - always available
print_progress() {
    echo "================================================================="
    echo "$1"
    echo "================================================================="
}

# =============================================================================
# FALLBACK FUNCTIONS (when libraries not available)
# =============================================================================

if [[ "$LIBS_LOADED" != "true" ]]; then
    # Minimal board detection
    detect_board_type() {
        if [[ -f /proc/device-tree/model ]]; then
            if grep -qi "raspberry" /proc/device-tree/model 2>/dev/null; then
                echo "raspberry_pi"
                return
            fi
        fi
        echo "generic_linux"
    }

    # Minimal serial console check
    detect_serial_console_enabled() {
        local cmdline_file="/boot/cmdline.txt"
        [[ -f /boot/firmware/cmdline.txt ]] && cmdline_file="/boot/firmware/cmdline.txt"
        [[ -f "$cmdline_file" ]] && grep -qE "console=(serial0|ttyAMA0|ttyS0)" "$cmdline_file"
    }

    # Minimal UART detection
    detect_uart_device() {
        if [[ -e /dev/serial0 ]]; then
            readlink -f /dev/serial0
        elif [[ -e /dev/ttyS0 ]]; then
            echo "/dev/ttyS0"
        elif [[ -e /dev/ttyAMA0 ]]; then
            echo "/dev/ttyAMA0"
        else
            echo "/dev/ttyS0"
        fi
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
SKIP_SERIAL_CHECK=false

show_help() {
    cat <<EOF
MAVLink-Anywhere Configuration Script v2.0.1

Usage: sudo ./configure_mavlink_router.sh [OPTIONS]

Modes:
  (default)         Interactive mode - checks prerequisites, prompts for settings
  --headless        Headless mode - no prompts, all settings via CLI
  --auto            Auto mode - auto-detect UART, use standard endpoints

Serial Input Options:
  --uart DEVICE     UART device path (e.g., /dev/ttyS0, /dev/ttyAMA0, /dev/ttyUSB0)
  --baud RATE       Baud rate (default: 57600)

UDP Input Options (no serial port needed):
  --input-type udp  Use UDP input instead of serial
  --input-address   UDP listen address (default: 0.0.0.0)
  --input-port      UDP listen port (default: 14550)

Endpoint Options:
  --endpoints LIST  Comma-separated endpoints (e.g., "127.0.0.1:14540,192.168.1.100:24550")
  --gcs-ip IP       GCS IP address (adds endpoint on port 24550)

Other:
  --skip-serial-check  Skip serial port prerequisite check
  -h, --help           Show this help message
  --debug              Enable debug output

Examples:
  # Interactive (checks serial prerequisites first)
  sudo ./configure_mavlink_router.sh

  # Auto-detect with GCS IP
  sudo ./configure_mavlink_router.sh --auto --gcs-ip 100.96.32.75

  # Headless with UART
  sudo ./configure_mavlink_router.sh --headless \\
      --uart /dev/ttyS0 --baud 57600 \\
      --endpoints "127.0.0.1:14540,127.0.0.1:14569,100.96.32.75:24550"

  # USB serial adapter (no boot config needed)
  sudo ./configure_mavlink_router.sh --auto --uart /dev/ttyUSB0

  # UDP input for SITL (no serial port needed)
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
        --skip-serial-check)
            SKIP_SERIAL_CHECK=true
            shift
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
# SERIAL PREREQUISITE CHECK FUNCTIONS
# =============================================================================

# Check if this is a USB serial device (doesn't need boot config)
is_usb_serial() {
    local device="$1"
    [[ "$device" == /dev/ttyUSB* ]] || [[ "$device" == /dev/ttyACM* ]]
}

# Auto-fix serial configuration on Raspberry Pi
auto_fix_serial_config() {
    local config_file="/boot/config.txt"
    local cmdline_file="/boot/cmdline.txt"

    # For newer Raspberry Pi OS
    [[ -f /boot/firmware/config.txt ]] && config_file="/boot/firmware/config.txt"
    [[ -f /boot/firmware/cmdline.txt ]] && cmdline_file="/boot/firmware/cmdline.txt"

    echo ""
    echo "Auto-fixing serial configuration..."
    echo ""

    # Backup original files
    local timestamp=$(date +%Y%m%d_%H%M%S)
    if [[ -f "$config_file" ]]; then
        sudo cp "$config_file" "${config_file}.backup_${timestamp}"
        echo "  Backed up: ${config_file}.backup_${timestamp}"
    fi
    if [[ -f "$cmdline_file" ]]; then
        sudo cp "$cmdline_file" "${cmdline_file}.backup_${timestamp}"
        echo "  Backed up: ${cmdline_file}.backup_${timestamp}"
    fi

    # Enable UART in config.txt
    if [[ -f "$config_file" ]]; then
        if ! grep -q "^enable_uart=1" "$config_file"; then
            echo "enable_uart=1" | sudo tee -a "$config_file" > /dev/null
            echo "  Added: enable_uart=1 to $config_file"
        else
            echo "  UART already enabled in $config_file"
        fi
    fi

    # Remove serial console from cmdline.txt
    if [[ -f "$cmdline_file" ]]; then
        if grep -qE "console=(serial0|ttyAMA0|ttyS0)" "$cmdline_file"; then
            sudo sed -i 's/console=serial0,[0-9]* //g' "$cmdline_file"
            sudo sed -i 's/console=ttyAMA0,[0-9]* //g' "$cmdline_file"
            sudo sed -i 's/console=ttyS0,[0-9]* //g' "$cmdline_file"
            echo "  Removed serial console from $cmdline_file"
        else
            echo "  Serial console already disabled in $cmdline_file"
        fi
    fi

    # Disable serial getty service
    sudo systemctl stop serial-getty@ttyS0.service 2>/dev/null || true
    sudo systemctl disable serial-getty@ttyS0.service 2>/dev/null || true
    sudo systemctl stop serial-getty@ttyAMA0.service 2>/dev/null || true
    sudo systemctl disable serial-getty@ttyAMA0.service 2>/dev/null || true
    echo "  Disabled serial getty services"

    echo ""
    echo "================================================================="
    echo "  REBOOT REQUIRED"
    echo "================================================================="
    echo ""
    echo "  Serial configuration has been updated."
    echo "  You MUST reboot for changes to take effect."
    echo ""
    echo "  After reboot, run this script again:"
    echo "    sudo ./configure_mavlink_router.sh"
    echo ""
    read -p "  Reboot now? (y/n): " reboot_choice
    if [[ "$reboot_choice" =~ ^[Yy] ]]; then
        echo ""
        echo "  Rebooting in 3 seconds..."
        sleep 3
        sudo reboot
    else
        echo ""
        echo "  Please reboot manually: sudo reboot"
        echo ""
        exit 0
    fi
}

# Display serial prerequisites menu
show_serial_prereq_menu() {
    local board_type="$1"
    local serial_console_enabled="$2"
    local uart_device="$3"

    echo ""
    echo "================================================================="
    echo "  Serial Port Configuration Required"
    echo "================================================================="
    echo ""

    if [[ "$serial_console_enabled" == "yes" ]]; then
        echo "  Issue: Serial console is enabled (blocking UART for MAVLink)"
    fi

    echo ""
    echo "  What would you like to do?"
    echo ""
    echo "    [1] Auto-fix configuration (Recommended for Raspberry Pi)"
    echo "        - Modifies /boot/config.txt and /boot/cmdline.txt"
    echo "        - Creates backup of original files"
    echo "        - REQUIRES REBOOT after fix"
    echo ""
    echo "    [2] Manual configuration"
    echo "        - Run 'sudo raspi-config'"
    echo "        - Interface Options -> Serial Port"
    echo "        - Login shell: NO, Hardware: YES"
    echo "        - Reboot and re-run this script"
    echo ""
    echo "    [3] Use USB serial instead (no reboot needed)"
    echo "        - Connect via USB-to-serial adapter"
    echo "        - Device: /dev/ttyUSB0 or /dev/ttyACM0"
    echo ""
    echo "    [4] Use UDP input (no serial port needed)"
    echo "        - For SITL simulation or network input"
    echo ""
    echo "    [5] Skip check and continue anyway"
    echo "        - May fail if serial is not properly configured"
    echo ""

    read -p "  Enter choice [1-5]: " choice

    case "$choice" in
        1)
            auto_fix_serial_config
            ;;
        2)
            echo ""
            echo "  Manual configuration steps:"
            echo "    1. Run: sudo raspi-config"
            echo "    2. Navigate to: Interface Options -> Serial Port"
            echo "    3. 'Login shell accessible over serial?' -> NO"
            echo "    4. 'Serial port hardware enabled?' -> YES"
            echo "    5. Finish and reboot"
            echo "    6. Re-run: sudo ./configure_mavlink_router.sh"
            echo ""
            exit 0
            ;;
        3)
            # Switch to USB serial
            for usb_dev in /dev/ttyUSB0 /dev/ttyACM0; do
                if [[ -e "$usb_dev" ]]; then
                    UART_DEVICE="$usb_dev"
                    echo ""
                    echo "  Using USB serial: $UART_DEVICE"
                    SKIP_SERIAL_CHECK=true
                    return 0
                fi
            done
            echo ""
            echo "  No USB serial device found. Please connect one and try again."
            exit 1
            ;;
        4)
            # Switch to UDP input
            INPUT_TYPE="udp"
            SKIP_SERIAL_CHECK=true
            echo ""
            echo "  Switched to UDP input mode"
            return 0
            ;;
        5)
            SKIP_SERIAL_CHECK=true
            echo ""
            echo "  Skipping serial check..."
            return 0
            ;;
        *)
            echo "  Invalid choice. Exiting."
            exit 1
            ;;
    esac
}

# Check serial prerequisites (main function)
check_serial_prerequisites() {
    # Skip if explicitly requested
    [[ "$SKIP_SERIAL_CHECK" == "true" ]] && return 0

    # Skip if using UDP input
    [[ "$INPUT_TYPE" == "udp" ]] && return 0

    # Skip if using USB serial (already specified)
    if [[ -n "$UART_DEVICE" ]] && is_usb_serial "$UART_DEVICE"; then
        return 0
    fi

    local board_type
    board_type=$(detect_board_type)

    # Only do detailed check on Raspberry Pi
    if [[ "$board_type" != "raspberry_pi" ]]; then
        echo "  Platform: $(echo "$board_type" | tr '_' ' ')"
        echo "  Note: Serial configuration is platform-specific."
        echo "        Please ensure UART is enabled on your board."
        echo ""
        return 0
    fi

    # Check for issues on Raspberry Pi
    local serial_console_enabled="no"
    if detect_serial_console_enabled; then
        serial_console_enabled="yes"
    fi

    local uart_device
    uart_device=$(detect_uart_device)

    # If serial console is enabled, we need to fix it
    if [[ "$serial_console_enabled" == "yes" ]]; then
        show_serial_prereq_menu "$board_type" "$serial_console_enabled" "$uart_device"
    fi

    return 0
}

# =============================================================================
# CONFIGURATION GENERATION (inline fallback)
# =============================================================================

generate_config_inline() {
    local input_type="$1"
    local uart_device="$2"
    local uart_baud="$3"
    local endpoints="$4"
    local input_address="$5"
    local input_port="$6"

    sudo mkdir -p /etc/mavlink-router
    sudo mkdir -p /etc/default

    # Generate main config
    if [[ "$input_type" == "uart" ]]; then
        sudo bash -c "cat > /etc/mavlink-router/main.conf" <<EOF
[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=${uart_device}
Baud=${uart_baud}
EOF
    else
        sudo bash -c "cat > /etc/mavlink-router/main.conf" <<EOF
[General]
TcpServerPort=5760
ReportStats=false

[UdpEndpoint input]
Mode=server
Address=${input_address}
Port=${input_port}
EOF
    fi

    # Add UDP endpoints
    IFS=',' read -r -a ENDPOINT_ARRAY <<< "${endpoints}"
    local INDEX=1
    for ENDPOINT in "${ENDPOINT_ARRAY[@]}"; do
        ENDPOINT=$(echo "$ENDPOINT" | tr -d ' ')
        local ADDRESS=$(echo ${ENDPOINT} | cut -d':' -f1)
        local PORT=$(echo ${ENDPOINT} | cut -d':' -f2)
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
UART_DEVICE=${uart_device}
UART_BAUD=${uart_baud}
UDP_ENDPOINTS="${endpoints}"
INPUT_TYPE=${input_type}
INPUT_ADDRESS=${input_address}
INPUT_PORT=${input_port}
EOF
}

setup_service_inline() {
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
}

# =============================================================================
# MODE ROUTING
# =============================================================================

if [[ "$HEADLESS" == "true" ]]; then
    # =========================================================================
    # HEADLESS MODE - No prompts, all settings from CLI
    # =========================================================================

    echo "================================================================="
    echo "MAVLink-Anywhere Configuration (Headless Mode)"
    echo "================================================================="

    # Validate required parameters based on input type
    if [[ "$INPUT_TYPE" == "uart" ]]; then
        if [[ -z "$UART_DEVICE" ]]; then
            echo "Error: --uart is required in headless mode for UART input"
            exit 1
        fi
        [[ -z "$UART_BAUD" ]] && UART_BAUD="57600"
    elif [[ "$INPUT_TYPE" == "udp" ]]; then
        [[ -z "$INPUT_PORT" ]] && INPUT_PORT="14550"
    else
        echo "Error: Invalid input type: $INPUT_TYPE (use 'uart' or 'udp')"
        exit 1
    fi

    if [[ -z "$ENDPOINTS" ]]; then
        echo "Error: --endpoints is required in headless mode"
        exit 1
    fi

    # Use library functions if available
    if [[ "$LIBS_LOADED" == "true" ]]; then
        if [[ "$INPUT_TYPE" == "uart" ]]; then
            configure_uart_headless "$UART_DEVICE" "$UART_BAUD" "$ENDPOINTS"
        else
            configure_udp_headless "$INPUT_ADDRESS" "$INPUT_PORT" "$ENDPOINTS"
        fi
        setup_service
    else
        generate_config_inline "$INPUT_TYPE" "$UART_DEVICE" "$UART_BAUD" "$ENDPOINTS" "$INPUT_ADDRESS" "$INPUT_PORT"
        setup_service_inline
    fi

    echo ""
    echo "Configuration complete!"
    echo "Config file: /etc/mavlink-router/main.conf"
    echo ""
    cat /etc/mavlink-router/main.conf

elif [[ "$AUTO_MODE" == "true" ]]; then
    # =========================================================================
    # AUTO MODE - Auto-detect settings, minimal prompts
    # =========================================================================

    echo "================================================================="
    echo "MAVLink-Anywhere Configuration (Auto Mode)"
    echo "================================================================="
    echo ""

    # Check serial prerequisites (may switch to USB or UDP)
    check_serial_prerequisites

    # Handle UDP mode if switched
    if [[ "$INPUT_TYPE" == "udp" ]]; then
        echo "Input: UDP on port ${INPUT_PORT}"
        [[ -z "$ENDPOINTS" ]] && ENDPOINTS="127.0.0.1:14540,127.0.0.1:14569,127.0.0.1:12550"

        if [[ -n "$GCS_IP" ]]; then
            ENDPOINTS="${ENDPOINTS},${GCS_IP}:24550"
        else
            read -p "Enter GCS IP address (or press Enter to skip): " GCS_IP
            [[ -n "$GCS_IP" ]] && ENDPOINTS="${ENDPOINTS},${GCS_IP}:24550"
        fi

        echo "Endpoints: $ENDPOINTS"

        if [[ "$LIBS_LOADED" == "true" ]]; then
            configure_udp_headless "$INPUT_ADDRESS" "$INPUT_PORT" "$ENDPOINTS"
            setup_service
        else
            generate_config_inline "udp" "" "" "$ENDPOINTS" "$INPUT_ADDRESS" "$INPUT_PORT"
            setup_service_inline
        fi
    else
        # UART mode
        # Auto-detect UART device if not specified
        if [[ -z "$UART_DEVICE" ]]; then
            if [[ "$LIBS_LOADED" == "true" ]]; then
                UART_DEVICE=$(detect_uart_device)
            else
                if [[ -e /dev/serial0 ]]; then
                    UART_DEVICE=$(readlink -f /dev/serial0)
                elif [[ -e /dev/ttyS0 ]]; then
                    UART_DEVICE="/dev/ttyS0"
                elif [[ -e /dev/ttyAMA0 ]]; then
                    UART_DEVICE="/dev/ttyAMA0"
                else
                    UART_DEVICE="/dev/ttyS0"
                fi
            fi
        fi
        echo "UART device: $UART_DEVICE"

        # Use default baud rate
        [[ -z "$UART_BAUD" ]] && UART_BAUD="57600"
        echo "Baud rate: $UART_BAUD"

        # Build endpoints list
        ENDPOINTS="127.0.0.1:14540,127.0.0.1:14569,127.0.0.1:12550"

        if [[ -n "$GCS_IP" ]]; then
            ENDPOINTS="${ENDPOINTS},${GCS_IP}:24550"
            echo "GCS endpoint: ${GCS_IP}:24550"
        else
            read -p "Enter GCS IP address (or press Enter to skip): " GCS_IP
            [[ -n "$GCS_IP" ]] && ENDPOINTS="${ENDPOINTS},${GCS_IP}:24550"
        fi

        echo "Endpoints: $ENDPOINTS"
        echo ""

        if [[ "$LIBS_LOADED" == "true" ]]; then
            configure_uart_headless "$UART_DEVICE" "$UART_BAUD" "$ENDPOINTS"
            setup_service
        else
            generate_config_inline "uart" "$UART_DEVICE" "$UART_BAUD" "$ENDPOINTS" "" ""
            setup_service_inline
        fi
    fi

    echo ""
    echo "================================================================="
    echo "Configuration complete!"
    echo "================================================================="
    echo "Config file: /etc/mavlink-router/main.conf"
    echo ""
    cat /etc/mavlink-router/main.conf
    echo ""
    echo "Check status: sudo systemctl status mavlink-router"
    echo "View logs:    sudo journalctl -u mavlink-router -f"

else
    # =========================================================================
    # INTERACTIVE MODE - Check prerequisites, then prompt for settings
    # =========================================================================

    echo "================================================================="
    echo "MAVLink-Anywhere Configuration"
    echo "================================================================="
    echo "GitHub: https://github.com/alireza787b/mavlink-anywhere"
    echo ""

    # Check serial prerequisites first
    check_serial_prerequisites

    # Handle UDP mode if switched during prereq check
    if [[ "$INPUT_TYPE" == "udp" ]]; then
        echo ""
        echo "Configuring UDP input mode..."
        echo ""

        read -p "Enter UDP listen port (default: 14550): " INPUT_PORT
        INPUT_PORT=${INPUT_PORT:-14550}

        read -p "Enter UDP endpoints (comma-separated, e.g., 127.0.0.1:14540,192.168.1.100:24550): " ENDPOINTS
        ENDPOINTS=${ENDPOINTS:-"127.0.0.1:14540,127.0.0.1:14569"}

        if [[ "$LIBS_LOADED" == "true" ]]; then
            configure_udp_headless "$INPUT_ADDRESS" "$INPUT_PORT" "$ENDPOINTS"
            setup_service
        else
            generate_config_inline "udp" "" "" "$ENDPOINTS" "$INPUT_ADDRESS" "$INPUT_PORT"
            setup_service_inline
        fi

        echo ""
        echo "================================================================="
        echo "Configuration complete!"
        echo "================================================================="
        cat /etc/mavlink-router/main.conf
        exit 0
    fi

    # Check if existing configuration exists
    CONFIG_FILE="/etc/mavlink-router/main.conf"
    if [[ -f "$CONFIG_FILE" ]]; then
        echo "Existing configuration found. Reading current settings..."
        source /etc/default/mavlink-router 2>/dev/null || true
        CFG_UART_DEVICE=${UART_DEVICE:-/dev/ttyS0}
        CFG_UART_BAUD=${UART_BAUD:-57600}
        CFG_UDP_ENDPOINTS=${UDP_ENDPOINTS:-0.0.0.0:14550}
    else
        # Use detected device as default
        CFG_UART_DEVICE=$(detect_uart_device)
        CFG_UART_BAUD="57600"
        CFG_UDP_ENDPOINTS="0.0.0.0:14550"
    fi

    # Override with CLI-specified device if provided
    [[ -n "$UART_DEVICE" ]] && CFG_UART_DEVICE="$UART_DEVICE"

    echo ""
    echo "Enter configuration (press Enter to use defaults):"
    echo ""

    read -p "UART device [${CFG_UART_DEVICE}]: " UART_DEVICE
    UART_DEVICE=${UART_DEVICE:-$CFG_UART_DEVICE}

    read -p "Baud rate [${CFG_UART_BAUD}]: " UART_BAUD
    UART_BAUD=${UART_BAUD:-$CFG_UART_BAUD}

    read -p "UDP endpoints (space-separated) [${CFG_UDP_ENDPOINTS}]: " UDP_ENDPOINTS
    UDP_ENDPOINTS=${UDP_ENDPOINTS:-$CFG_UDP_ENDPOINTS}

    echo ""
    print_progress "Creating configuration..."

    # Create environment file
    sudo mkdir -p /etc/default
    sudo bash -c "cat > /etc/default/mavlink-router" <<EOF
UART_DEVICE=${UART_DEVICE}
UART_BAUD=${UART_BAUD}
UDP_ENDPOINTS="${UDP_ENDPOINTS}"
EOF

    # Create configuration file
    sudo mkdir -p /etc/mavlink-router
    sudo bash -c "cat > /etc/mavlink-router/main.conf" <<EOF
[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=${UART_DEVICE}
Baud=${UART_BAUD}
EOF

    # Add UDP endpoints
    IFS=' ' read -r -a ENDPOINT_ARRAY <<< "${UDP_ENDPOINTS}"
    INDEX=1
    for ENDPOINT in "${ENDPOINT_ARRAY[@]}"; do
        sudo bash -c "cat >> /etc/mavlink-router/main.conf" <<EOF

[UdpEndpoint udp${INDEX}]
Mode=normal
Address=$(echo ${ENDPOINT} | cut -d':' -f1)
Port=$(echo ${ENDPOINT} | cut -d':' -f2)
EOF
        INDEX=$((INDEX+1))
    done

    # Setup service
    print_progress "Setting up systemd service..."

    sudo systemctl stop mavlink-router 2>/dev/null || true

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
    sudo systemctl start mavlink-router

    # Print success
    echo ""
    echo "================================================================="
    echo "Configuration complete!"
    echo "================================================================="
    echo ""
    echo "Configuration file: /etc/mavlink-router/main.conf"
    echo ""
    cat /etc/mavlink-router/main.conf
    echo ""
    echo "================================================================="
    echo "Quick Commands"
    echo "================================================================="
    echo ""
    echo "  Status & Logs:"
    echo "    sudo systemctl status mavlink-router"
    echo "    sudo journalctl -u mavlink-router -f"
    echo ""
    echo "  Service Control:"
    echo "    sudo systemctl restart mavlink-router"
    echo "    sudo systemctl stop mavlink-router"
    echo ""
    echo "  Reconfigure:"
    echo "    ./configure_mavlink_router.sh              # Interactive"
    echo "    ./configure_mavlink_router.sh --headless \\"
    echo "        --endpoints \"127.0.0.1:14550,192.168.1.100:14550\""
    echo ""
    echo "  Edit Config Directly:"
    echo "    sudo nano /etc/mavlink-router/main.conf"
    echo "    sudo systemctl restart mavlink-router"
    echo ""
    if [[ -f "${SCRIPT_DIR}/mavlink-router-cli.sh" ]]; then
    echo "  CLI Helper (all-in-one):"
    echo "    ./mavlink-router-cli.sh status"
    echo "    ./mavlink-router-cli.sh logs"
    echo "    ./mavlink-router-cli.sh endpoints"
    echo ""
    fi
    echo "================================================================="
    echo "Connect QGroundControl to this device's IP on the configured UDP ports."
    echo "================================================================="
fi
