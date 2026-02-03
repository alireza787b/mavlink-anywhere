#!/bin/bash
# =============================================================================
# MAVLink-Anywhere Library: Hardware Detection
# =============================================================================
# Version: 2.0.0
# Description: UART/USB detection, board identification, serial status checking
# Author: Alireza Ghaderi
# GitHub: https://github.com/alireza787b/mavlink-anywhere
# =============================================================================

# Prevent double-sourcing
[[ -n "${_MAVLINK_DETECT_LOADED:-}" ]] && return 0
_MAVLINK_DETECT_LOADED=1

# Source common library if not already loaded
if [[ -z "${_MAVLINK_COMMON_LOADED:-}" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
fi

# =============================================================================
# BOARD DETECTION
# =============================================================================

# Detect board type
# Returns: raspberry_pi, jetson, generic_linux, unknown
detect_board_type() {
    local board_type="unknown"

    # Check for Raspberry Pi
    if [[ -f /proc/device-tree/model ]]; then
        local model
        model=$(cat /proc/device-tree/model 2>/dev/null | tr -d '\0')

        if echo "$model" | grep -qi "raspberry"; then
            board_type="raspberry_pi"
        elif echo "$model" | grep -qi "jetson"; then
            board_type="jetson"
        fi
    fi

    # Fallback checks
    if [[ "$board_type" == "unknown" ]]; then
        if [[ -f /etc/rpi-issue ]] || [[ -d /opt/vc ]]; then
            board_type="raspberry_pi"
        elif [[ -d /etc/nv_tegra_release ]] || [[ -f /etc/nv_tegra_release ]]; then
            board_type="jetson"
        elif [[ -f /etc/os-release ]]; then
            board_type="generic_linux"
        fi
    fi

    echo "$board_type"
}

# Detect Raspberry Pi model
# Returns: pi5, pi4, pi3, pizero2, pizero, pi_other, not_pi
detect_rpi_model() {
    if [[ ! -f /proc/device-tree/model ]]; then
        echo "not_pi"
        return
    fi

    local model
    model=$(cat /proc/device-tree/model 2>/dev/null | tr -d '\0')

    if echo "$model" | grep -qi "raspberry pi 5"; then
        echo "pi5"
    elif echo "$model" | grep -qi "raspberry pi 4"; then
        echo "pi4"
    elif echo "$model" | grep -qi "raspberry pi 3"; then
        echo "pi3"
    elif echo "$model" | grep -qi "zero 2"; then
        echo "pizero2"
    elif echo "$model" | grep -qi "zero"; then
        echo "pizero"
    elif echo "$model" | grep -qi "raspberry"; then
        echo "pi_other"
    else
        echo "not_pi"
    fi
}

# Get human-readable board description
get_board_description() {
    local board_type
    board_type=$(detect_board_type)

    case "$board_type" in
        raspberry_pi)
            local model
            model=$(detect_rpi_model)
            case "$model" in
                pi5) echo "Raspberry Pi 5" ;;
                pi4) echo "Raspberry Pi 4" ;;
                pi3) echo "Raspberry Pi 3" ;;
                pizero2) echo "Raspberry Pi Zero 2 W" ;;
                pizero) echo "Raspberry Pi Zero" ;;
                *) echo "Raspberry Pi (Other)" ;;
            esac
            ;;
        jetson)
            echo "NVIDIA Jetson"
            ;;
        generic_linux)
            echo "Generic Linux"
            ;;
        *)
            echo "Unknown Board"
            ;;
    esac
}

# Get system architecture
get_system_architecture() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        aarch64|arm64) echo "arm64" ;;
        armv7l|armhf) echo "armhf" ;;
        x86_64) echo "x86_64" ;;
        i686|i386) echo "i386" ;;
        *) echo "$arch" ;;
    esac
}

# =============================================================================
# UART DETECTION
# =============================================================================

# Detect available UART devices
# Returns list of available UART device paths
detect_uart_devices() {
    local devices=()

    # Check for standard serial devices
    for device in /dev/ttyAMA* /dev/ttyS* /dev/serial*; do
        if [[ -e "$device" ]]; then
            # Resolve symlinks
            local resolved
            resolved=$(readlink -f "$device" 2>/dev/null || echo "$device")
            # Avoid duplicates
            if [[ ! " ${devices[*]} " =~ " ${resolved} " ]]; then
                devices+=("$resolved")
            fi
        fi
    done

    # Sort and return
    printf '%s\n' "${devices[@]}" | sort -u
}

# Detect USB serial devices
detect_usb_serial_devices() {
    local devices=()

    for device in /dev/ttyUSB* /dev/ttyACM*; do
        if [[ -e "$device" ]]; then
            devices+=("$device")
        fi
    done

    printf '%s\n' "${devices[@]}" | sort
}

# Auto-detect the best UART device for the current board
# Priority: /dev/serial0 > board-specific > first available
detect_uart_device() {
    local rpi_model
    rpi_model=$(detect_rpi_model)

    # Priority 1: Use /dev/serial0 symlink if available
    if [[ -e /dev/serial0 ]]; then
        local resolved
        resolved=$(readlink -f /dev/serial0)
        if [[ -e "$resolved" ]]; then
            echo "$resolved"
            return 0
        fi
    fi

    # Priority 2: Board-specific defaults
    case "$rpi_model" in
        pi5)
            # Pi 5 uses /dev/ttyAMA0 for GPIO UART
            if [[ -e /dev/ttyAMA0 ]]; then
                echo "/dev/ttyAMA0"
                return 0
            fi
            ;;
        pi4|pizero2)
            # Pi 4 and Zero 2 typically use /dev/ttyS0 (mini UART) when BT enabled
            # or /dev/ttyAMA0 when BT disabled
            if [[ -e /dev/ttyAMA0 ]]; then
                echo "/dev/ttyAMA0"
                return 0
            elif [[ -e /dev/ttyS0 ]]; then
                echo "/dev/ttyS0"
                return 0
            fi
            ;;
        pi3|pizero|pi_other)
            # Older Pi models typically use /dev/ttyS0
            if [[ -e /dev/ttyS0 ]]; then
                echo "/dev/ttyS0"
                return 0
            fi
            ;;
    esac

    # Priority 3: Try common devices
    for device in /dev/ttyS0 /dev/ttyAMA0; do
        if [[ -e "$device" ]]; then
            echo "$device"
            return 0
        fi
    done

    # Priority 4: Check for USB serial
    for device in /dev/ttyUSB0 /dev/ttyACM0; do
        if [[ -e "$device" ]]; then
            echo "$device"
            return 0
        fi
    done

    # Fallback: return default
    echo "/dev/ttyS0"
    return 1
}

# Validate a UART device exists and is accessible
validate_uart_device() {
    local device="$1"

    # Check device exists
    if [[ ! -e "$device" ]]; then
        ma_log_error "UART device not found: $device"
        return 1
    fi

    # Check device is a character device
    if [[ ! -c "$device" ]]; then
        ma_log_error "Not a character device: $device"
        return 1
    fi

    # Check read/write permissions
    if [[ ! -r "$device" ]] || [[ ! -w "$device" ]]; then
        ma_log_warn "Permission denied for $device - may need dialout group"
        return 2
    fi

    return 0
}

# =============================================================================
# SERIAL CONSOLE DETECTION
# =============================================================================

# Check if serial console is enabled (blocking UART for MAVLink)
detect_serial_console_enabled() {
    local cmdline_file="/boot/cmdline.txt"

    # For newer Raspberry Pi OS with /boot/firmware
    if [[ -f /boot/firmware/cmdline.txt ]]; then
        cmdline_file="/boot/firmware/cmdline.txt"
    fi

    if [[ -f "$cmdline_file" ]]; then
        if grep -qE "console=(serial0|ttyAMA0|ttyS0)" "$cmdline_file"; then
            return 0  # Serial console IS enabled
        fi
    fi

    return 1  # Serial console NOT enabled
}

# Check if UART is enabled in boot config
detect_uart_enabled() {
    local config_file="/boot/config.txt"

    # For newer Raspberry Pi OS with /boot/firmware
    if [[ -f /boot/firmware/config.txt ]]; then
        config_file="/boot/firmware/config.txt"
    fi

    if [[ -f "$config_file" ]]; then
        # Check for enable_uart=1 (not commented out)
        if grep -qE "^enable_uart=1" "$config_file"; then
            return 0  # UART IS enabled
        fi
    fi

    # UART might be enabled by default on some systems
    # Check if device actually exists
    if [[ -e /dev/serial0 ]] || [[ -e /dev/ttyS0 ]] || [[ -e /dev/ttyAMA0 ]]; then
        return 0
    fi

    return 1  # UART NOT explicitly enabled
}

# Check if user is in dialout group
check_dialout_group() {
    local user="${1:-$(whoami)}"

    if groups "$user" 2>/dev/null | grep -q '\bdialout\b'; then
        return 0
    fi

    return 1
}

# =============================================================================
# COMPREHENSIVE SERIAL STATUS CHECK
# =============================================================================

# Perform comprehensive serial port status check
# Returns JSON-like status string
check_serial_status() {
    local board_type board_desc rpi_model arch
    local uart_enabled serial_console uart_device uart_valid dialout_ok

    board_type=$(detect_board_type)
    board_desc=$(get_board_description)
    rpi_model=$(detect_rpi_model)
    arch=$(get_system_architecture)

    # Check UART status
    if detect_uart_enabled; then
        uart_enabled="yes"
    else
        uart_enabled="no"
    fi

    # Check serial console
    if detect_serial_console_enabled; then
        serial_console="enabled"
    else
        serial_console="disabled"
    fi

    # Detect UART device
    uart_device=$(detect_uart_device)
    if validate_uart_device "$uart_device" 2>/dev/null; then
        uart_valid="yes"
    else
        uart_valid="no"
    fi

    # Check dialout group
    if check_dialout_group; then
        dialout_ok="yes"
    else
        dialout_ok="no"
    fi

    # Output status (values must be quoted for eval)
    cat <<EOF
board_type="$board_type"
board_description="$board_desc"
rpi_model="$rpi_model"
architecture="$arch"
uart_enabled="$uart_enabled"
serial_console="$serial_console"
uart_device="$uart_device"
uart_valid="$uart_valid"
dialout_group="$dialout_ok"
EOF
}

# Display formatted serial status
display_serial_status() {
    local status
    status=$(check_serial_status)

    local board_desc uart_enabled serial_console uart_device uart_valid dialout_ok

    # Parse status
    eval "$status"

    ma_print_box "Serial Port Configuration Status"
    ma_print_box_line ""
    ma_print_box_line "Hardware:"
    ma_print_box_line "  Board:           $board_description"
    ma_print_box_line "  Architecture:    $architecture"
    ma_print_box_line ""
    ma_print_box_line "Serial Status:"

    # UART enabled status
    if [[ "$uart_enabled" == "yes" ]]; then
        ma_print_box_line "  UART enabled:       $(echo -e "${MA_GREEN}✓ Yes${MA_NC}")"
    else
        ma_print_box_line "  UART enabled:       $(echo -e "${MA_RED}✗ No${MA_NC}")"
    fi

    # Serial console status (we want it disabled)
    if [[ "$serial_console" == "disabled" ]]; then
        ma_print_box_line "  Serial console:     $(echo -e "${MA_GREEN}✗ Disabled (good)${MA_NC}")"
    else
        ma_print_box_line "  Serial console:     $(echo -e "${MA_YELLOW}⚠ ENABLED (blocking UART!)${MA_NC}")"
    fi

    # Device availability
    if [[ "$uart_valid" == "yes" ]]; then
        ma_print_box_line "  Device available:   $(echo -e "${MA_GREEN}✓ $uart_device${MA_NC}")"
    else
        ma_print_box_line "  Device available:   $(echo -e "${MA_RED}✗ $uart_device (not accessible)${MA_NC}")"
    fi

    # Dialout group
    if [[ "$dialout_ok" == "yes" ]]; then
        ma_print_box_line "  Permissions:        $(echo -e "${MA_GREEN}✓ User in dialout group${MA_NC}")"
    else
        ma_print_box_line "  Permissions:        $(echo -e "${MA_YELLOW}⚠ User not in dialout group${MA_NC}")"
    fi

    ma_print_box_line ""
    ma_print_box_end
}

# Check if serial configuration has issues that need fixing
serial_needs_configuration() {
    local board_type
    board_type=$(detect_board_type)

    # Only check on Raspberry Pi
    if [[ "$board_type" != "raspberry_pi" ]]; then
        return 1  # Non-Pi boards - assume OK or manual config
    fi

    # Check for issues
    if detect_serial_console_enabled; then
        return 0  # Needs fixing - serial console enabled
    fi

    if ! detect_uart_enabled; then
        return 0  # Needs fixing - UART not enabled
    fi

    local uart_device
    uart_device=$(detect_uart_device)
    if ! validate_uart_device "$uart_device" 2>/dev/null; then
        return 0  # Needs fixing - device not accessible
    fi

    return 1  # No issues detected
}

# =============================================================================
# INPUT METHOD DETECTION
# =============================================================================

# Detect all available input methods
detect_available_inputs() {
    local inputs=()

    # Check for UART
    local uart_device
    uart_device=$(detect_uart_device)
    if [[ -e "$uart_device" ]]; then
        inputs+=("uart:$uart_device")
    fi

    # Check for USB serial
    for device in /dev/ttyUSB* /dev/ttyACM*; do
        if [[ -e "$device" ]]; then
            inputs+=("usb:$device")
        fi
    done

    # UDP is always available
    inputs+=("udp:0.0.0.0:14550")

    printf '%s\n' "${inputs[@]}"
}
