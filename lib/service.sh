#!/bin/bash
# =============================================================================
# MAVLink-Anywhere Library: Service Management
# =============================================================================
# Version: 2.0.0
# Description: systemd service management for mavlink-router
# Author: Alireza Ghaderi
# GitHub: https://github.com/alireza787b/mavlink-anywhere
# =============================================================================

# Prevent double-sourcing
[[ -n "${_MAVLINK_SERVICE_LOADED:-}" ]] && return 0
_MAVLINK_SERVICE_LOADED=1

# Source common library if not already loaded
if [[ -z "${_MAVLINK_COMMON_LOADED:-}" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
fi

# =============================================================================
# SERVICE FILE MANAGEMENT
# =============================================================================

readonly MAVLINK_SERVICE_FILE="/etc/systemd/system/mavlink-router.service"

# Generate systemd service file content
generate_service_file() {
    cat <<'EOF'
[Unit]
Description=MAVLink Router Service
Documentation=https://github.com/alireza787b/mavlink-anywhere
After=network.target

[Service]
Type=simple
EnvironmentFile=/etc/default/mavlink-router
ExecStart=/usr/bin/mavlink-routerd -c /etc/mavlink-router/main.conf
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/mavlink-router
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
}

# Install systemd service file
install_service_file() {
    ma_log_step "Installing systemd service file..."

    # Backup existing service file if present
    if [[ -f "$MAVLINK_SERVICE_FILE" ]]; then
        ma_backup_file "$MAVLINK_SERVICE_FILE"
    fi

    # Write new service file
    generate_service_file > "$MAVLINK_SERVICE_FILE"

    if [[ $? -ne 0 ]]; then
        ma_log_error "Failed to write service file"
        return 1
    fi

    # Set permissions
    chmod 644 "$MAVLINK_SERVICE_FILE"

    # Reload systemd
    systemctl daemon-reload

    ma_log_success "Service file installed: $MAVLINK_SERVICE_FILE"
    return 0
}

# =============================================================================
# SERVICE STATUS FUNCTIONS
# =============================================================================

# Check if mavlink-router binary is installed
is_mavlink_router_installed() {
    if ma_command_exists mavlink-routerd; then
        return 0
    fi

    # Check common installation paths
    for path in /usr/bin/mavlink-routerd /usr/local/bin/mavlink-routerd; do
        if [[ -x "$path" ]]; then
            return 0
        fi
    done

    return 1
}

# Check if service is active (running)
is_service_active() {
    systemctl is-active --quiet "${MAVLINK_ROUTER_SERVICE}" 2>/dev/null
}

# Check if service is enabled (starts at boot)
is_service_enabled() {
    systemctl is-enabled --quiet "${MAVLINK_ROUTER_SERVICE}" 2>/dev/null
}

# Check if service file exists
service_file_exists() {
    [[ -f "$MAVLINK_SERVICE_FILE" ]] || systemctl list-unit-files | grep -q "mavlink-router.service"
}

# Get service status as string
get_service_status() {
    if ! service_file_exists; then
        echo "not_installed"
        return
    fi

    if is_service_active; then
        echo "running"
    elif is_service_enabled; then
        echo "stopped_enabled"
    else
        echo "stopped_disabled"
    fi
}

# Get service uptime
get_service_uptime() {
    if is_service_active; then
        systemctl show "${MAVLINK_ROUTER_SERVICE}" --property=ActiveEnterTimestamp | cut -d= -f2
    else
        echo "not running"
    fi
}

# =============================================================================
# SERVICE CONTROL FUNCTIONS
# =============================================================================

# Start the service
start_service() {
    ma_log_step "Starting mavlink-router service..."

    if ! service_file_exists; then
        ma_log_error "Service not installed. Run installation first."
        return 1
    fi

    systemctl start "${MAVLINK_ROUTER_SERVICE}"
    local result=$?

    if [[ $result -eq 0 ]]; then
        sleep 1  # Give service time to start
        if is_service_active; then
            ma_log_success "Service started successfully"
            return 0
        fi
    fi

    ma_log_error "Failed to start service"
    return 1
}

# Stop the service
stop_service() {
    ma_log_step "Stopping mavlink-router service..."

    if ! is_service_active; then
        ma_log_info "Service is not running"
        return 0
    fi

    systemctl stop "${MAVLINK_ROUTER_SERVICE}"

    if ! is_service_active; then
        ma_log_success "Service stopped"
        return 0
    fi

    ma_log_error "Failed to stop service"
    return 1
}

# Restart the service
restart_service() {
    ma_log_step "Restarting mavlink-router service..."

    if ! service_file_exists; then
        ma_log_error "Service not installed"
        return 1
    fi

    systemctl restart "${MAVLINK_ROUTER_SERVICE}"
    local result=$?

    sleep 1  # Give service time to restart

    if [[ $result -eq 0 ]] && is_service_active; then
        ma_log_success "Service restarted successfully"
        return 0
    fi

    ma_log_error "Failed to restart service"
    return 1
}

# Enable the service (start at boot)
enable_service() {
    ma_log_step "Enabling mavlink-router service..."

    if ! service_file_exists; then
        ma_log_error "Service not installed"
        return 1
    fi

    systemctl enable "${MAVLINK_ROUTER_SERVICE}" 2>/dev/null

    if is_service_enabled; then
        ma_log_success "Service enabled for automatic start"
        return 0
    fi

    ma_log_error "Failed to enable service"
    return 1
}

# Disable the service
disable_service() {
    ma_log_step "Disabling mavlink-router service..."

    systemctl disable "${MAVLINK_ROUTER_SERVICE}" 2>/dev/null

    if ! is_service_enabled; then
        ma_log_success "Service disabled"
        return 0
    fi

    ma_log_error "Failed to disable service"
    return 1
}

# =============================================================================
# SERVICE SETUP AND TEARDOWN
# =============================================================================

# Full service setup: install, enable, start
setup_service() {
    ma_log_step "Setting up mavlink-router service..."

    # Stop existing service if running
    if is_service_active; then
        stop_service
    fi

    # Install service file
    install_service_file || return 1

    # Enable service
    enable_service || return 1

    # Start service
    start_service || return 1

    ma_log_success "Service setup complete"
    return 0
}

# Full service removal
remove_service() {
    ma_log_step "Removing mavlink-router service..."

    # Stop service
    stop_service 2>/dev/null

    # Disable service
    disable_service 2>/dev/null

    # Remove service file
    if [[ -f "$MAVLINK_SERVICE_FILE" ]]; then
        rm -f "$MAVLINK_SERVICE_FILE"
        ma_log_info "Removed: $MAVLINK_SERVICE_FILE"
    fi

    # Reload systemd
    systemctl daemon-reload

    ma_log_success "Service removed"
    return 0
}

# =============================================================================
# STATUS DISPLAY
# =============================================================================

# Display comprehensive service status
display_service_status() {
    ma_print_box "MAVLink Router Service Status"

    # Binary status
    if is_mavlink_router_installed; then
        local version
        version=$(mavlink-routerd --version 2>&1 | head -1 || echo "unknown")
        ma_print_box_line "Binary:      $(echo -e "${MA_GREEN}✓ Installed${MA_NC}") ($version)"
    else
        ma_print_box_line "Binary:      $(echo -e "${MA_RED}✗ Not installed${MA_NC}")"
    fi

    # Service file status
    if service_file_exists; then
        ma_print_box_line "Service:     $(echo -e "${MA_GREEN}✓ Configured${MA_NC}")"
    else
        ma_print_box_line "Service:     $(echo -e "${MA_YELLOW}○ Not configured${MA_NC}")"
    fi

    # Running status
    local status
    status=$(get_service_status)
    case "$status" in
        running)
            ma_print_box_line "Status:      $(echo -e "${MA_GREEN}● Running${MA_NC}")"
            local uptime
            uptime=$(get_service_uptime)
            ma_print_box_line "Since:       $uptime"
            ;;
        stopped_enabled)
            ma_print_box_line "Status:      $(echo -e "${MA_YELLOW}○ Stopped (enabled)${MA_NC}")"
            ;;
        stopped_disabled)
            ma_print_box_line "Status:      $(echo -e "${MA_RED}○ Stopped (disabled)${MA_NC}")"
            ;;
        not_installed)
            ma_print_box_line "Status:      $(echo -e "${MA_DIM}─ Not installed${MA_NC}")"
            ;;
    esac

    # Config file status
    if [[ -f "$MAVLINK_ROUTER_CONFIG_FILE" ]]; then
        ma_print_box_line "Config:      $(echo -e "${MA_GREEN}✓${MA_NC}") $MAVLINK_ROUTER_CONFIG_FILE"
    else
        ma_print_box_line "Config:      $(echo -e "${MA_YELLOW}○${MA_NC}") Not found"
    fi

    ma_print_box_line ""
    ma_print_box_end
}

# Show service logs
show_service_logs() {
    local lines="${1:-20}"

    echo ""
    ma_log_info "Recent mavlink-router logs (last $lines lines):"
    echo ""

    journalctl -u "${MAVLINK_ROUTER_SERVICE}" -n "$lines" --no-pager 2>/dev/null || \
        ma_log_warn "Unable to retrieve logs"
}

# Follow service logs in real-time
follow_service_logs() {
    ma_log_info "Following mavlink-router logs (Ctrl+C to stop)..."
    echo ""

    journalctl -u "${MAVLINK_ROUTER_SERVICE}" -f 2>/dev/null || \
        ma_log_warn "Unable to follow logs"
}

# =============================================================================
# SERVICE VERIFICATION
# =============================================================================

# Verify service is working correctly
verify_service() {
    local checks_passed=0
    local checks_total=4

    ma_print_section "Service Verification"

    # Check 1: Binary installed
    if is_mavlink_router_installed; then
        ma_log_success "mavlink-routerd binary found"
        ((checks_passed++))
    else
        ma_log_error "mavlink-routerd binary not found"
    fi

    # Check 2: Service file exists
    if service_file_exists; then
        ma_log_success "Service file configured"
        ((checks_passed++))
    else
        ma_log_error "Service file not found"
    fi

    # Check 3: Config file exists and valid
    if [[ -f "$MAVLINK_ROUTER_CONFIG_FILE" ]]; then
        # Source config library for validation
        if [[ -z "${_MAVLINK_CONFIG_LOADED:-}" ]]; then
            source "$(dirname "${BASH_SOURCE[0]}")/config.sh"
        fi
        if validate_config_file "$MAVLINK_ROUTER_CONFIG_FILE" 2>/dev/null; then
            ((checks_passed++))
        fi
    else
        ma_log_error "Configuration file not found"
    fi

    # Check 4: Service running
    if is_service_active; then
        ma_log_success "Service is running"
        ((checks_passed++))
    else
        ma_log_warn "Service is not running"
    fi

    echo ""
    ma_log_info "Verification: $checks_passed/$checks_total checks passed"

    [[ $checks_passed -eq $checks_total ]]
}
