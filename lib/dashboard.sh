#!/bin/bash
# =============================================================================
# MAVLink-Anywhere Library: Dashboard Management
# =============================================================================
# Version: 3.0.11
# Description: Install, configure, and manage the web dashboard binary
# Author: Alireza Ghaderi
# GitHub: https://github.com/alireza787b/mavlink-anywhere
# =============================================================================

# Prevent double-sourcing
[[ -n "${_MAVLINK_DASHBOARD_LOADED:-}" ]] && return 0
_MAVLINK_DASHBOARD_LOADED=1

# Source common library if not already loaded
if [[ -z "${_MAVLINK_COMMON_LOADED:-}" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/common.sh"
fi

# =============================================================================
# DASHBOARD BINARY MANAGEMENT
# =============================================================================

# Map uname -m to release binary suffix
_dashboard_arch_suffix() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        aarch64|arm64)   echo "linux-arm64" ;;
        armv7l|armhf)    echo "linux-arm6"  ;;   # arm6 covers armv6+armv7
        armv6l)          echo "linux-arm6"  ;;
        x86_64)          echo "linux-amd64" ;;
        *)
            ma_log_warn "Unsupported architecture for dashboard: $arch"
            echo ""
            return 1
            ;;
    esac
}

# Check if dashboard binary is installed and executable
is_dashboard_installed() {
    local binary="${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
    [[ -x "$binary" ]]
}

# Get installed dashboard version (returns "" if not installed)
get_dashboard_version() {
    local binary="${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
    if [[ -x "$binary" ]]; then
        "$binary" --version 2>/dev/null | head -1
    fi
}

dashboard_binary_is_current() {
    local installed_version
    installed_version=$(get_dashboard_version)
    [[ -n "$installed_version" && "$installed_version" == *" v${MAVLINK_ANYWHERE_VERSION} "* ]]
}

dashboard_binary_looks_valid() {
    local binary_path="$1"
    local magic=""

    if ma_command_exists file; then
        file "$binary_path" 2>/dev/null | grep -qE "ELF|executable"
        return $?
    fi

    ma_log_debug "'file' utility not available; falling back to ELF header validation"
    magic=$(dd if="$binary_path" bs=1 count=4 2>/dev/null | od -An -tx1 2>/dev/null | tr -d '[:space:]')
    [[ "$magic" == "7f454c46" ]]
}

dashboard_env_has_auth() {
    [[ -f "$DASHBOARD_ENV_FILE" ]] && grep -q '^MAVLINK_ANYWHERE_DASHBOARD_PASSWORD_BCRYPT=' "$DASHBOARD_ENV_FILE"
}

env_quote() {
    printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

generate_dashboard_password() {
    python3 - <<'PY'
import secrets
alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789-_."
print("".join(secrets.choice(alphabet) for _ in range(24)))
PY
}

hash_dashboard_password() {
    local password="$1"
    if ! "${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}" --help 2>&1 | grep -q -- '--hash-password'; then
        ma_log_warn "Installed dashboard binary does not support password hashing; rebuilding from local source."
        build_dashboard_binary_from_source || {
            ma_log_error "Cannot configure dashboard auth because the dashboard binary cannot hash passwords."
            return 1
        }
    fi
    printf '%s' "$password" | "${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}" --hash-password
}

configure_dashboard_auth() {
    local listen_addr="${1:-127.0.0.1:${DASHBOARD_PORT}}"
    local auth_user="${DASHBOARD_AUTH_USER:-admin}"
    local auth_hash="${DASHBOARD_AUTH_HASH:-}"
    local password=""
    local should_generate="${DASHBOARD_GENERATE_PASSWORD:-false}"
    local remote_dashboard=false

    if ! dashboard_is_local_only "$listen_addr"; then
        remote_dashboard=true
    fi

    if [[ "${DASHBOARD_DISABLE_AUTH:-false}" == "true" ]]; then
        ma_log_warn "Dashboard browser auth explicitly disabled. Use only on an isolated trusted network."
        return 0
    fi

    if [[ -z "$auth_hash" && -n "${DASHBOARD_AUTH_PASSWORD_FILE:-}" ]]; then
        if [[ ! -r "$DASHBOARD_AUTH_PASSWORD_FILE" ]]; then
            ma_log_error "Dashboard password file is not readable: $DASHBOARD_AUTH_PASSWORD_FILE"
            return 1
        fi
        password="$(tr -d '\r\n' < "$DASHBOARD_AUTH_PASSWORD_FILE")"
        if [[ -z "$password" ]]; then
            ma_log_error "Dashboard password file is empty."
            return 1
        fi
        auth_hash="$(hash_dashboard_password "$password")"
    fi

    if [[ -z "$auth_hash" && "$should_generate" != "true" && "$remote_dashboard" == "true" && ! dashboard_env_has_auth ]]; then
        should_generate=true
    fi

    if [[ -z "$auth_hash" && "$should_generate" == "true" ]]; then
        password="$(generate_dashboard_password)"
        auth_hash="$(hash_dashboard_password "$password")"
    fi

    if [[ -z "$auth_hash" ]]; then
        if dashboard_env_has_auth; then
            ma_log_info "Dashboard browser auth already configured in $DASHBOARD_ENV_FILE"
        elif [[ "$remote_dashboard" == "true" ]]; then
            ma_log_warn "Remote dashboard has no browser auth configured. Re-run with --dashboard-generate-password or --dashboard-auth-password-file."
        else
            ma_log_info "Dashboard browser auth not configured for local-only listen address."
        fi
        return 0
    fi

    ma_ensure_dir "$(dirname "$DASHBOARD_ENV_FILE")"
    {
        echo "# MAVLink Anywhere dashboard browser auth"
        echo "# Generated by configure_mavlink_router.sh. Store only password hashes here."
        printf 'MAVLINK_ANYWHERE_DASHBOARD_USER='
        env_quote "$auth_user"
        printf '\nMAVLINK_ANYWHERE_DASHBOARD_PASSWORD_BCRYPT='
        env_quote "$auth_hash"
        printf '\n'
    } > "$DASHBOARD_ENV_FILE"
    chmod 600 "$DASHBOARD_ENV_FILE"
    ma_log_success "Dashboard browser auth configured in $DASHBOARD_ENV_FILE"

    if [[ -n "$password" ]]; then
        echo ""
        echo "Dashboard login username: $auth_user"
        echo "Dashboard generated password: $password"
        echo "Store this password now; it is not recoverable from $DASHBOARD_ENV_FILE."
        echo ""
    else
        ma_log_info "Dashboard login username: $auth_user"
    fi
}

# Download and install the dashboard binary from GitHub Releases.
# This is a best-effort operation — failure does NOT block mavlink-router setup.
install_dashboard_binary() {
    local version="${1:-latest}"

    local suffix
    suffix=$(_dashboard_arch_suffix) || return 1

    ma_log_step "Installing dashboard binary (${suffix})..."

    local download_url
    if [[ "$version" == "latest" ]]; then
        download_url="${DASHBOARD_RELEASES_URL}/latest/download/${DASHBOARD_BINARY_NAME}-${suffix}"
    else
        download_url="${DASHBOARD_RELEASES_URL}/download/${version}/${DASHBOARD_BINARY_NAME}-${suffix}"
    fi

    ma_log_debug "Download URL: $download_url"

    # Download to temp file first
    local tmp_file
    tmp_file=$(mktemp /tmp/mavlink-anywhere-dashboard.XXXXXX)

    if curl -fsSL --connect-timeout 15 --max-time 120 -o "$tmp_file" "$download_url" 2>/dev/null; then
        # Verify it's a real binary (not an HTML error page)
        if dashboard_binary_looks_valid "$tmp_file"; then
            ma_ensure_dir "$DASHBOARD_INSTALL_DIR"
            mv "$tmp_file" "${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
            chmod +x "${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
            ma_log_success "Dashboard binary installed: ${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
            return 0
        else
            ma_log_warn "Downloaded file is not a valid dashboard binary (release may not exist yet)"
            rm -f "$tmp_file"
            return 1
        fi
    else
        ma_log_warn "Failed to download dashboard binary (no internet or release not found)"
        rm -f "$tmp_file"
        return 1
    fi
}

# Build the dashboard binary from the local source tree.
# This is used as a fallback when a release asset is unavailable for the host
# architecture. It intentionally avoids the caller's default Go cache so it can
# run in minimal environments.
build_dashboard_binary_from_source() {
    local repo_root dashboard_dir tmp_bin cache_dir version build_time
    repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    dashboard_dir="${repo_root}/dashboard"

    if [[ ! -f "${dashboard_dir}/go.mod" ]]; then
        ma_log_warn "Dashboard source tree not found at ${dashboard_dir}"
        return 1
    fi

    if ! ma_command_exists go; then
        ma_log_warn "Go toolchain not found — cannot build dashboard from source"
        return 1
    fi

    ma_log_step "Building dashboard binary from local source..."

    tmp_bin=$(mktemp /tmp/mavlink-anywhere-dashboard.XXXXXX)
    cache_dir=$(mktemp -d /tmp/mavlink-anywhere-go-cache.XXXXXX)
    version="${MAVLINK_ANYWHERE_VERSION}"
    build_time="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

    if (
        cd "$dashboard_dir" && \
        env GOCACHE="$cache_dir" CGO_ENABLED=0 \
            go build -ldflags "-s -w -X main.Version=${version} -X main.BuildTime=${build_time}" \
            -o "$tmp_bin" ./cmd/
    ); then
        ma_ensure_dir "$DASHBOARD_INSTALL_DIR"
        mv "$tmp_bin" "${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
        chmod +x "${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
        rm -rf "$cache_dir"
        ma_log_success "Dashboard binary built from source: ${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
        return 0
    fi

    rm -f "$tmp_bin"
    rm -rf "$cache_dir"
    ma_log_warn "Local dashboard source build failed"
    return 1
}

# =============================================================================
# SYSTEMD SERVICE MANAGEMENT
# =============================================================================

readonly DASHBOARD_SERVICE_FILE="/etc/systemd/system/${DASHBOARD_SERVICE}.service"

# Generate the dashboard systemd service file
generate_dashboard_service() {
    local listen_addr="${1:-127.0.0.1:${DASHBOARD_PORT}}"

    cat <<EOF
[Unit]
Description=MAVLink Anywhere Web Dashboard
Documentation=https://github.com/alireza787b/mavlink-anywhere
After=network.target mavlink-router.service
Wants=mavlink-router.service

[Service]
Type=simple
EnvironmentFile=-${DASHBOARD_ENV_FILE}
ExecStart=${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME} --listen ${listen_addr}
Restart=on-failure
RestartSec=5
MemoryMax=30M
MemoryHigh=20M
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/etc/mavlink-router /etc/default /etc/mavlink-anywhere

[Install]
WantedBy=multi-user.target
EOF
}

# Install and enable the dashboard systemd service
setup_dashboard_service() {
    local listen_addr="${1:-127.0.0.1:${DASHBOARD_PORT}}"

    if ! is_dashboard_installed; then
        ma_log_warn "Dashboard binary not found — skipping service setup"
        return 1
    fi

    ma_log_step "Setting up dashboard service..."
    ma_ensure_dir "$MAVLINK_ROUTER_CONFIG_DIR"
    configure_dashboard_auth "$listen_addr" || return 1

    # Write service file
    generate_dashboard_service "$listen_addr" > "$DASHBOARD_SERVICE_FILE"
    chmod 644 "$DASHBOARD_SERVICE_FILE"

    # Reload and enable
    systemctl daemon-reload
    systemctl enable "$DASHBOARD_SERVICE" 2>/dev/null

    # Always restart so service file changes and binary updates take effect
    systemctl restart "$DASHBOARD_SERVICE" 2>/dev/null

    if systemctl is-active --quiet "$DASHBOARD_SERVICE" 2>/dev/null; then
        ma_log_success "Dashboard service started on ${listen_addr}"
    else
        ma_log_info "Dashboard service enabled (will start on next boot)"
    fi

    return 0
}

dashboard_is_local_only() {
    local listen_addr="${1:-127.0.0.1:${DASHBOARD_PORT}}"
    [[ "$listen_addr" =~ ^(127\.0\.0\.1|localhost|\[::1\]): ]]
}

# Remove the dashboard service (leaves binary in place)
remove_dashboard_service() {
    ma_log_step "Removing dashboard service..."

    systemctl stop "$DASHBOARD_SERVICE" 2>/dev/null
    systemctl disable "$DASHBOARD_SERVICE" 2>/dev/null

    if [[ -f "$DASHBOARD_SERVICE_FILE" ]]; then
        rm -f "$DASHBOARD_SERVICE_FILE"
    fi

    systemctl daemon-reload
    ma_log_success "Dashboard service removed"
}

# Check dashboard service status
is_dashboard_running() {
    systemctl is-active --quiet "$DASHBOARD_SERVICE" 2>/dev/null
}

# =============================================================================
# HIGH-LEVEL INSTALL/UNINSTALL
# =============================================================================

# Full dashboard setup: download binary + install service.
# Returns 0 on success, 1 on failure (non-fatal).
install_dashboard() {
    local listen_addr="${1:-127.0.0.1:${DASHBOARD_PORT}}"

    echo ""
    ma_log_step "Setting up web dashboard..."

    # Try to download binary
    if ! is_dashboard_installed; then
        if ! install_dashboard_binary; then
            if ! build_dashboard_binary_from_source; then
                ma_log_warn "Dashboard install failed — skipping dashboard setup"
                ma_log_info "You can install it later: sudo ${DASHBOARD_INSTALL_DIR}/configure_mavlink_router.sh --install-dashboard"
                ma_log_info "If release assets are unavailable, install Go and rerun to build the dashboard locally."
                return 1
            fi
        fi
    elif ! dashboard_binary_is_current; then
        ma_log_info "Updating dashboard binary: $(get_dashboard_version) -> v${MAVLINK_ANYWHERE_VERSION}"
        if ! install_dashboard_binary "v${MAVLINK_ANYWHERE_VERSION}"; then
            ma_log_warn "Dashboard binary update failed — keeping existing version"
        fi
    else
        ma_log_info "Dashboard binary already installed: $(get_dashboard_version)"
    fi

    # Setup service
    setup_dashboard_service "$listen_addr"

    echo ""
    if dashboard_is_local_only "$listen_addr"; then
        ma_log_success "Dashboard available locally at: http://${listen_addr}"
        ma_log_info "Use an SSH tunnel or re-run with --dashboard-listen 0.0.0.0:${DASHBOARD_PORT} to expose it on the network."
    else
        ma_log_success "Dashboard listening on: http://${listen_addr}"
        ma_log_info "If a firewall is enabled, allow TCP ${DASHBOARD_PORT} or use a VPN/SSH tunnel."
    fi
    return 0
}

# Full dashboard removal: stop service + optionally remove binary.
uninstall_dashboard() {
    local remove_binary="${1:-false}"

    remove_dashboard_service

    if [[ "$remove_binary" == "true" ]]; then
        local binary="${DASHBOARD_INSTALL_DIR}/${DASHBOARD_BINARY_NAME}"
        if [[ -f "$binary" ]]; then
            rm -f "$binary"
            ma_log_success "Dashboard binary removed"
        fi
    fi
}
