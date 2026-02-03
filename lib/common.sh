#!/bin/bash
# =============================================================================
# MAVLink-Anywhere Library: Common Utilities
# =============================================================================
# Version: 2.0.1
# Description: Core utilities - colors, logging, shared functions
# Author: Alireza Ghaderi
# GitHub: https://github.com/alireza787b/mavlink-anywhere
# =============================================================================

# Prevent double-sourcing
[[ -n "${_MAVLINK_COMMON_LOADED:-}" ]] && return 0
_MAVLINK_COMMON_LOADED=1

# =============================================================================
# CONSTANTS
# =============================================================================

readonly MAVLINK_ANYWHERE_VERSION="2.0.1"
readonly MAVLINK_ROUTER_CONFIG_DIR="/etc/mavlink-router"
readonly MAVLINK_ROUTER_CONFIG_FILE="${MAVLINK_ROUTER_CONFIG_DIR}/main.conf"
readonly MAVLINK_ROUTER_ENV_FILE="/etc/default/mavlink-router"
readonly MAVLINK_ROUTER_SERVICE="mavlink-router"

# Default configuration values
readonly DEFAULT_UART_DEVICE="/dev/ttyS0"
readonly DEFAULT_UART_BAUD="57600"
readonly DEFAULT_UDP_PORT="14550"

# Standard MDS endpoints
readonly MDS_ENDPOINT_MAVSDK="127.0.0.1:14540"
readonly MDS_ENDPOINT_MAVLINK2REST="127.0.0.1:14569"
readonly MDS_ENDPOINT_LOCAL="127.0.0.1:12550"
readonly MDS_ENDPOINT_GCS_PORT="24550"

# =============================================================================
# TERMINAL COLORS
# =============================================================================

# Check if stdout is a terminal
if [[ -t 1 ]]; then
    readonly MA_RED='\033[0;31m'
    readonly MA_GREEN='\033[0;32m'
    readonly MA_YELLOW='\033[1;33m'
    readonly MA_BLUE='\033[0;34m'
    readonly MA_MAGENTA='\033[0;35m'
    readonly MA_CYAN='\033[0;36m'
    readonly MA_WHITE='\033[1;37m'
    readonly MA_BOLD='\033[1m'
    readonly MA_DIM='\033[2m'
    readonly MA_NC='\033[0m'  # No Color
    readonly MA_CHECK="${MA_GREEN}[✓]${MA_NC}"
    readonly MA_CROSS="${MA_RED}[✗]${MA_NC}"
    readonly MA_ARROW="${MA_CYAN}[→]${MA_NC}"
    readonly MA_WARN="${MA_YELLOW}[!]${MA_NC}"
    readonly MA_INFO="${MA_BLUE}[i]${MA_NC}"
else
    readonly MA_RED=''
    readonly MA_GREEN=''
    readonly MA_YELLOW=''
    readonly MA_BLUE=''
    readonly MA_MAGENTA=''
    readonly MA_CYAN=''
    readonly MA_WHITE=''
    readonly MA_BOLD=''
    readonly MA_DIM=''
    readonly MA_NC=''
    readonly MA_CHECK='[OK]'
    readonly MA_CROSS='[FAIL]'
    readonly MA_ARROW='[->]'
    readonly MA_WARN='[!]'
    readonly MA_INFO='[i]'
fi

# =============================================================================
# LOGGING FUNCTIONS
# =============================================================================

ma_log_info() {
    echo -e "  ${MA_INFO} $1"
}

ma_log_success() {
    echo -e "  ${MA_CHECK} $1"
}

ma_log_warn() {
    echo -e "  ${MA_WARN} ${MA_YELLOW}$1${MA_NC}"
}

ma_log_error() {
    echo -e "  ${MA_CROSS} ${MA_RED}$1${MA_NC}"
}

ma_log_step() {
    echo -e "  ${MA_ARROW} $1"
}

ma_log_debug() {
    if [[ "${MA_DEBUG:-false}" == "true" ]]; then
        echo -e "  ${MA_DIM}[DEBUG] $1${MA_NC}"
    fi
}

# =============================================================================
# DISPLAY FUNCTIONS
# =============================================================================

ma_print_header() {
    echo "================================================================="
    echo "MavlinkAnywhere: $1"
    echo "Author: Alireza Ghaderi"
    echo "GitHub: https://github.com/alireza787b/mavlink-anywhere"
    echo "Version: ${MAVLINK_ANYWHERE_VERSION}"
    echo "================================================================="
}

ma_print_progress() {
    echo "================================================================="
    echo "$1"
    echo "================================================================="
}

ma_print_section() {
    local title="$1"
    echo ""
    echo -e "  ${MA_BOLD}${title}${MA_NC}"
    echo -e "  ${MA_DIM}$(printf '%.0s─' {1..60})${MA_NC}"
}

ma_print_box() {
    local title="$1"
    local width=76

    echo ""
    echo -e "${MA_CYAN}┌$(printf '%.0s─' $(seq 1 $width))┐${MA_NC}"
    echo -e "${MA_CYAN}│${MA_NC}  ${MA_WHITE}${title}${MA_NC}$(printf '%*s' $((width - ${#title} - 3)) '')${MA_CYAN}│${MA_NC}"
    echo -e "${MA_CYAN}├$(printf '%.0s─' $(seq 1 $width))┤${MA_NC}"
}

ma_print_box_line() {
    local line="$1"
    local width=76
    printf "${MA_CYAN}│${MA_NC} %-$((width - 2))s ${MA_CYAN}│${MA_NC}\n" "$line"
}

ma_print_box_end() {
    local width=76
    echo -e "${MA_CYAN}└$(printf '%.0s─' $(seq 1 $width))┘${MA_NC}"
    echo ""
}

# =============================================================================
# UTILITY FUNCTIONS
# =============================================================================

# Check if a command exists
ma_command_exists() {
    command -v "$1" &>/dev/null
}

# Check if running as root
ma_check_root() {
    if [[ $EUID -ne 0 ]]; then
        return 1
    fi
    return 0
}

# Validate IP address format
ma_validate_ip() {
    local ip="$1"
    local regex='^([0-9]{1,3}\.){3}[0-9]{1,3}$'

    if [[ ! "$ip" =~ $regex ]]; then
        return 1
    fi

    IFS='.' read -ra octets <<< "$ip"
    for octet in "${octets[@]}"; do
        if [[ "$octet" -gt 255 ]]; then
            return 1
        fi
    done

    return 0
}

# Validate port number
ma_validate_port() {
    local port="$1"

    if [[ ! "$port" =~ ^[0-9]+$ ]]; then
        return 1
    fi

    if [[ "$port" -lt 1 || "$port" -gt 65535 ]]; then
        return 1
    fi

    return 0
}

# Validate endpoint format (IP:PORT)
ma_validate_endpoint() {
    local endpoint="$1"

    if [[ ! "$endpoint" =~ ^.+:[0-9]+$ ]]; then
        return 1
    fi

    local ip="${endpoint%:*}"
    local port="${endpoint##*:}"

    ma_validate_ip "$ip" && ma_validate_port "$port"
}

# Backup a file with timestamp
ma_backup_file() {
    local file="$1"
    if [[ -f "$file" ]]; then
        local backup="${file}.bak.$(date +%Y%m%d_%H%M%S)"
        cp "$file" "$backup"
        ma_log_debug "Backed up: $file -> $backup"
        echo "$backup"
    fi
}

# Create directory if it doesn't exist
ma_ensure_dir() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir"
        ma_log_debug "Created directory: $dir"
    fi
}

# Prompt for input with default value
ma_prompt_input() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"

    local input
    read -p "  $prompt [$default]: " input

    eval "$var_name=\"${input:-$default}\""
}

# Prompt for yes/no confirmation
ma_confirm() {
    local prompt="$1"
    local default="${2:-y}"

    local yn
    if [[ "$default" == "y" ]]; then
        read -p "  $prompt [Y/n]: " yn
        yn=${yn:-y}
    else
        read -p "  $prompt [y/N]: " yn
        yn=${yn:-n}
    fi

    [[ "${yn,,}" == "y" || "${yn,,}" == "yes" ]]
}

# Parse comma-separated endpoints into array
ma_parse_endpoints() {
    local endpoints_str="$1"
    local -n result_array=$2

    IFS=',' read -ra result_array <<< "$endpoints_str"

    # Trim whitespace from each endpoint
    for i in "${!result_array[@]}"; do
        result_array[$i]=$(echo "${result_array[$i]}" | tr -d ' ')
    done
}

# Build default MDS endpoints string
ma_get_mds_endpoints() {
    local gcs_ip="${1:-}"
    local endpoints="${MDS_ENDPOINT_MAVSDK},${MDS_ENDPOINT_MAVLINK2REST},${MDS_ENDPOINT_LOCAL}"

    if [[ -n "$gcs_ip" ]]; then
        endpoints="${endpoints},${gcs_ip}:${MDS_ENDPOINT_GCS_PORT}"
    fi

    echo "$endpoints"
}

# =============================================================================
# EXPORT LIBRARY PATH
# =============================================================================

# Determine script directory for sourcing other libraries
if [[ -z "${MAVLINK_ANYWHERE_LIB_DIR:-}" ]]; then
    MAVLINK_ANYWHERE_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi
export MAVLINK_ANYWHERE_LIB_DIR
