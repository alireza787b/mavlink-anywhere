#!/usr/bin/env bash
set -euo pipefail

# Beginner-first operator CLI for MAVLink Anywhere.

resolve_script_dir() {
    local source_path="${BASH_SOURCE[0]}"
    while [[ -L "$source_path" ]]; do
        local source_dir
        source_dir="$(cd -P "$(dirname "$source_path")" && pwd)"
        source_path="$(readlink "$source_path")"
        [[ "$source_path" != /* ]] && source_path="${source_dir}/${source_path}"
    done
    cd -P "$(dirname "$source_path")" && pwd
}

SCRIPT_DIR="$(resolve_script_dir)"
SERVICE_NAME="${MLA_ROUTER_SERVICE:-mavlink-router}"
CONFIG_FILE="${MLA_CONFIG_FILE:-/etc/mavlink-router/main.conf}"
ROUTER_ENV_FILE="${MLA_ROUTER_ENV_FILE:-/etc/default/mavlink-router}"
CONFIG_TOOL="${MLA_CONFIG_TOOL:-${SCRIPT_DIR}/scripts/mavlink_config.py}"
DASHBOARD_SERVICE="${MLA_DASHBOARD_SERVICE:-mavlink-anywhere-dashboard}"
DASHBOARD_SERVICE_FILE="${MLA_DASHBOARD_SERVICE_FILE:-/etc/systemd/system/${DASHBOARD_SERVICE}.service}"
DASHBOARD_ENV_FILE="${MLA_DASHBOARD_ENV_FILE:-/etc/mavlink-anywhere/dashboard.env}"
DASHBOARD_BINARY="${MLA_DASHBOARD_BINARY:-/opt/mavlink-anywhere/mavlink-anywhere}"
DEFAULT_DASHBOARD_PORT="9070"

die() {
    echo "Error: $*" >&2
    exit 1
}

need_root() {
    [[ ${EUID:-$(id -u)} -eq 0 ]] || die "Run this command with sudo."
}

need_config_tool() {
    [[ -f "$CONFIG_TOOL" ]] || die "Config helper not found: $CONFIG_TOOL. Update the MAVLink Anywhere checkout first."
    command -v python3 >/dev/null || die "python3 is required."
}

service_exists() {
    systemctl cat "$1" >/dev/null 2>&1
}

service_running() {
    systemctl is-active --quiet "$1" 2>/dev/null
}

show_help() {
    cat <<'EOF'
MAVLink Anywhere (mla)

Usage: mla COMMAND

Start here:
  mla status                    Check the router, dashboard, input and routes
  mla help endpoint             Add, edit, disable or remove MAVLink routes
  mla help input                Change the Pixhawk/flight-data input
  mla help dashboard            Open, hide, stop or secure the web UI
  mla help config               Safely edit the raw router config

Router:
  mla logs                      Follow router logs (Ctrl+C exits)
  mla restart                   Restart the router
  mla start | stop              Start or stop the router

Run changing commands with sudo, for example:
  sudo mla endpoint add qgc 192.168.1.50 14550
EOF
}

show_endpoint_help() {
    cat <<'EOF'
Manage MAVLink routes

  mla endpoint list
  sudo mla endpoint add NAME IP PORT [normal|server]
  sudo mla endpoint edit NAME IP PORT [normal|server]
  sudo mla endpoint disable NAME
  sudo mla endpoint enable NAME
  sudo mla endpoint remove NAME

Examples:
  sudo mla endpoint add qgc 192.168.1.50 14550
  sudo mla endpoint add mavsdk 127.0.0.1 14540
  sudo mla endpoint edit qgc 100.80.10.20 24550
  sudo mla endpoint disable qgc
  sudo mla endpoint enable qgc
  sudo mla endpoint remove qgc

normal sends MAVLink to that IP. server listens on this device.
Every change is validated, backed up, and applied to a running router.
EOF
}

show_input_help() {
    cat <<'EOF'
Manage the flight-data input

  mla input show
  sudo mla input uart DEVICE [BAUD]
  sudo mla input udp ADDRESS PORT

Examples:
  sudo mla input uart /dev/serial0 57600
  sudo mla input uart /dev/ttyUSB0 115200
  sudo mla input udp 0.0.0.0 14560

Input means where MAVLink first arrives from Pixhawk, another router, or SITL.
EOF
}

show_config_help() {
    cat <<EOF
Raw configuration

  mla config show               Print $CONFIG_FILE
  mla config path               Print the config and environment paths
  mla config check              Validate without changing anything
  sudo mla config edit          Open the raw file safely in \$EDITOR or nano

Raw edit creates a backup, validates the result, syncs the companion env file,
and applies it only if valid. A failed router restart restores the old files.
EOF
}

show_dashboard_help() {
    cat <<'EOF'
Manage the web dashboard

  mla dashboard status
  sudo mla dashboard expose [PORT]       Allow access from your network
  sudo mla dashboard hide                Revert expose; localhost only
  sudo mla dashboard off                 Stop and disable the UI completely
  sudo mla dashboard on                  Enable and start the UI

Browser password:
  mla dashboard password status
  sudo mla dashboard password reset [USER]
  sudo mla dashboard password generate [USER]

Machine API token:
  mla dashboard token status
  sudo mla dashboard token create
  sudo mla dashboard token rotate
  sudo mla dashboard token remove

Important: hide reverses network exposure. off stops the UI but does not change
its saved address, so use hide before off when you want a local-only next start.
EOF
}

show_help_topic() {
    case "${1:-}" in
        endpoint|endpoints|route|routes) show_endpoint_help ;;
        input|source) show_input_help ;;
        dashboard|ui|auth|token|password) show_dashboard_help ;;
        config|raw) show_config_help ;;
        "") show_help ;;
        *) die "Unknown help topic: $1. Try: mla help" ;;
    esac
}

dashboard_env_get() {
    local key="$1"
    [[ -f "$DASHBOARD_ENV_FILE" ]] || return 0
    python3 - "$DASHBOARD_ENV_FILE" "$key" <<'PY'
import shlex
import sys
from pathlib import Path

path, wanted = Path(sys.argv[1]), sys.argv[2]
for raw in path.read_text(encoding="utf-8").splitlines():
    line = raw.strip()
    if not line or line.startswith("#") or "=" not in line:
        continue
    key, value = line.split("=", 1)
    if key.strip() == wanted:
        try:
            parts = shlex.split(value, posix=True)
            print(parts[0] if parts else "")
        except ValueError:
            print(value.strip().strip("'\""))
        break
PY
}

dashboard_env_set() {
    local key="$1" value="$2"
    python3 - "$DASHBOARD_ENV_FILE" "$key" "$value" <<'PY'
import os
import shlex
import sys
import tempfile
from pathlib import Path

path, wanted, value = Path(sys.argv[1]), sys.argv[2], sys.argv[3]
path.parent.mkdir(parents=True, exist_ok=True)
lines = path.read_text(encoding="utf-8").splitlines() if path.exists() else ["# MAVLink Anywhere dashboard settings"]
updated = []
found = False
for line in lines:
    if not line.lstrip().startswith("#") and "=" in line and line.split("=", 1)[0].strip() == wanted:
        if not found:
            updated.append(f"{wanted}={shlex.quote(value)}")
            found = True
        continue
    updated.append(line)
if not found:
    updated.append(f"{wanted}={shlex.quote(value)}")
fd, temp_name = tempfile.mkstemp(prefix=".dashboard.env.", dir=path.parent)
try:
    with os.fdopen(fd, "w", encoding="utf-8") as handle:
        handle.write("\n".join(updated).rstrip() + "\n")
    os.chmod(temp_name, 0o600)
    os.replace(temp_name, path)
finally:
    if os.path.exists(temp_name):
        os.unlink(temp_name)
PY
}

dashboard_env_unset() {
    local key="$1"
    [[ -f "$DASHBOARD_ENV_FILE" ]] || return 0
    python3 - "$DASHBOARD_ENV_FILE" "$key" <<'PY'
import os
import sys
import tempfile
from pathlib import Path

path, wanted = Path(sys.argv[1]), sys.argv[2]
lines = path.read_text(encoding="utf-8").splitlines()
updated = [line for line in lines if line.lstrip().startswith("#") or "=" not in line or line.split("=", 1)[0].strip() != wanted]
fd, temp_name = tempfile.mkstemp(prefix=".dashboard.env.", dir=path.parent)
try:
    with os.fdopen(fd, "w", encoding="utf-8") as handle:
        handle.write("\n".join(updated).rstrip() + "\n")
    os.chmod(temp_name, 0o600)
    os.replace(temp_name, path)
finally:
    if os.path.exists(temp_name):
        os.unlink(temp_name)
PY
}

dashboard_current_listen() {
    local value
    value="$(dashboard_env_get MAVLINK_ANYWHERE_DASHBOARD_LISTEN)"
    if [[ -z "$value" && -f "$DASHBOARD_SERVICE_FILE" ]]; then
        value="$(sed -n 's/.*--listen[ =]\([^ ]*\).*/\1/p' "$DASHBOARD_SERVICE_FILE" | head -1)"
    fi
    printf '%s\n' "${value:-127.0.0.1:${DEFAULT_DASHBOARD_PORT}}"
}

dashboard_port() {
    local listen
    listen="$(dashboard_current_listen)"
    printf '%s\n' "${listen##*:}"
}

dashboard_has_password() {
    [[ -n "$(dashboard_env_get MAVLINK_ANYWHERE_DASHBOARD_USER)" ]] &&
        [[ -n "$(dashboard_env_get MAVLINK_ANYWHERE_DASHBOARD_PASSWORD_BCRYPT)" ]]
}

dashboard_has_token() {
    [[ -n "$(dashboard_env_get MAVLINK_ANYWHERE_API_TOKEN)" ]]
}

dashboard_require_installed() {
    [[ -x "$DASHBOARD_BINARY" ]] || die "Dashboard is not installed. Run: sudo ./configure_mavlink_router.sh --install-dashboard"
    service_exists "$DASHBOARD_SERVICE" || die "Dashboard service is not installed. Run: sudo ./configure_mavlink_router.sh --install-dashboard"
}

random_password() {
    python3 - <<'PY'
import secrets
alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789-_."
print("".join(secrets.choice(alphabet) for _ in range(24)))
PY
}

random_token() {
    python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(32))
PY
}

set_dashboard_password() {
    local user="$1" password="$2" hash
    [[ -x "$DASHBOARD_BINARY" ]] || die "Dashboard binary cannot hash the password: $DASHBOARD_BINARY"
    hash="$(printf '%s' "$password" | "$DASHBOARD_BINARY" --hash-password)"
    [[ "$hash" == \$2* ]] || die "Dashboard password hashing failed."
    dashboard_env_set MAVLINK_ANYWHERE_DASHBOARD_USER "$user"
    dashboard_env_set MAVLINK_ANYWHERE_DASHBOARD_PASSWORD_BCRYPT "$hash"
    dashboard_env_unset MAVLINK_ANYWHERE_ALLOW_UNAUTHENTICATED_MUTATIONS
}

dashboard_password_generate() {
    need_root
    dashboard_require_installed
    local user="${1:-admin}" password
    password="$(random_password)"
    set_dashboard_password "$user" "$password"
    service_running "$DASHBOARD_SERVICE" && systemctl restart "$DASHBOARD_SERVICE"
    echo "Dashboard username: $user"
    echo "Dashboard password: $password"
    echo "Save this password now; MLA will not print it again."
}

dashboard_password_reset() {
    need_root
    dashboard_require_installed
    local user="${1:-admin}" password confirm
    [[ -t 0 ]] || die "Password reset needs an interactive terminal. Use 'password generate' for a generated password."
    read -r -s -p "New dashboard password: " password
    echo
    read -r -s -p "Confirm dashboard password: " confirm
    echo
    [[ -n "$password" ]] || die "Password cannot be empty."
    [[ "$password" == "$confirm" ]] || die "Passwords did not match."
    set_dashboard_password "$user" "$password"
    service_running "$DASHBOARD_SERVICE" && systemctl restart "$DASHBOARD_SERVICE"
    echo "Dashboard password reset for user: $user"
}

dashboard_status() {
    local listen state startup access auth token
    listen="$(dashboard_current_listen)"
    if service_running "$DASHBOARD_SERVICE"; then state="running"; else state="stopped"; fi
    if systemctl is-enabled --quiet "$DASHBOARD_SERVICE" 2>/dev/null; then startup="enabled"; else startup="disabled"; fi
    if [[ "$listen" == 127.0.0.1:* || "$listen" == localhost:* || "$listen" == \[::1\]:* ]]; then
        access="local only"
    else
        access="network exposed"
    fi
    if dashboard_has_password; then auth="configured"; else auth="not configured"; fi
    if dashboard_has_token; then token="configured"; else token="not configured"; fi
    cat <<EOF
Dashboard:       $state ($startup at boot)
Access:          $access
Listen address:  $listen
Browser login:   $auth
Machine token:   $token
EOF
    if [[ "$state" == "running" ]]; then
        if [[ "$access" == "local only" ]]; then
            echo "Open locally:    http://$listen"
        else
            local device_ip
            device_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
            [[ -n "$device_ip" ]] && echo "Open on LAN:     http://${device_ip}:$(dashboard_port)"
        fi
    fi
}

dashboard_expose() {
    need_root
    dashboard_require_installed
    local port="${1:-$(dashboard_port)}"
    [[ "$port" =~ ^[0-9]+$ ]] && (( port >= 1 && port <= 65535 )) || die "Port must be between 1 and 65535."
    if ! dashboard_has_password; then
        local password
        password="$(random_password)"
        set_dashboard_password admin "$password"
        echo "Dashboard username: admin"
        echo "Dashboard password: $password"
        echo "Save this password now; MLA will not print it again."
        echo
    fi
    dashboard_env_set MAVLINK_ANYWHERE_DASHBOARD_LISTEN "0.0.0.0:${port}"
    systemctl enable --now "$DASHBOARD_SERVICE" >/dev/null
    systemctl restart "$DASHBOARD_SERVICE"
    echo "Dashboard exposed on this device's network addresses, TCP $port."
    echo "Revert anytime: sudo mla dashboard hide"
}

dashboard_hide() {
    need_root
    dashboard_require_installed
    local port
    port="$(dashboard_port)"
    dashboard_env_set MAVLINK_ANYWHERE_DASHBOARD_LISTEN "127.0.0.1:${port}"
    if service_running "$DASHBOARD_SERVICE"; then
        systemctl restart "$DASHBOARD_SERVICE"
        echo "Dashboard is now local-only at http://127.0.0.1:${port}"
    else
        echo "Local-only access saved. The dashboard remains off."
    fi
}

dashboard_token() {
    local action="${1:-status}" token
    case "$action" in
        status)
            if dashboard_has_token; then echo "Machine API token: configured"; else echo "Machine API token: not configured"; fi
            ;;
        create)
            need_root
            if dashboard_has_token; then
                die "A token already exists. Use 'sudo mla dashboard token rotate' to replace it."
            fi
            token="$(random_token)"
            dashboard_env_set MAVLINK_ANYWHERE_API_TOKEN "$token"
            service_running "$DASHBOARD_SERVICE" && systemctl restart "$DASHBOARD_SERVICE"
            echo "Machine API token: $token"
            echo "Save this token now; MLA will not print it again."
            ;;
        rotate)
            need_root
            token="$(random_token)"
            dashboard_env_set MAVLINK_ANYWHERE_API_TOKEN "$token"
            service_running "$DASHBOARD_SERVICE" && systemctl restart "$DASHBOARD_SERVICE"
            echo "New machine API token: $token"
            echo "Update every client that used the old token."
            ;;
        remove)
            need_root
            dashboard_env_unset MAVLINK_ANYWHERE_API_TOKEN
            service_running "$DASHBOARD_SERVICE" && systemctl restart "$DASHBOARD_SERVICE"
            echo "Machine API token removed."
            ;;
        *) die "Unknown token command: $action. Try: mla help dashboard" ;;
    esac
}

dashboard_command() {
    local action="${1:-status}"
    shift || true
    case "$action" in
        status) dashboard_status ;;
        expose) dashboard_expose "${1:-}" ;;
        hide|local) dashboard_hide ;;
        off|disable)
            need_root
            dashboard_require_installed
            systemctl disable --now "$DASHBOARD_SERVICE"
            echo "Dashboard stopped and disabled. MAVLink routing is still running."
            ;;
        on|enable)
            need_root
            dashboard_require_installed
            systemctl enable --now "$DASHBOARD_SERVICE"
            dashboard_status
            ;;
        password)
            case "${1:-status}" in
                status) if dashboard_has_password; then echo "Browser login: configured"; else echo "Browser login: not configured"; fi ;;
                reset) dashboard_password_reset "${2:-admin}" ;;
                generate) dashboard_password_generate "${2:-admin}" ;;
                *) die "Unknown password command. Try: mla help dashboard" ;;
            esac
            ;;
        token) dashboard_token "${1:-status}" ;;
        help|-h|--help) show_dashboard_help ;;
        *) die "Unknown dashboard command: $action. Try: mla help dashboard" ;;
    esac
}

backup_to_temp() {
    local source="$1" destination="$2"
    if [[ -f "$source" ]]; then
        cp -p "$source" "$destination"
        return 0
    fi
    return 1
}

restore_optional() {
    local backup="$1" target="$2" existed="$3"
    if [[ "$existed" == "true" ]]; then
        cp -p "$backup" "$target"
    else
        rm -f "$target"
    fi
}

apply_router_mutation() {
    need_root
    need_config_tool
    [[ -f "$CONFIG_FILE" ]] || die "Router config not found: $CONFIG_FILE"
    local temp_dir config_copy env_copy config_existed=true env_existed=false was_running=false
    temp_dir="$(mktemp -d /tmp/mla-change.XXXXXX)"
    config_copy="${temp_dir}/main.conf"
    env_copy="${temp_dir}/mavlink-router.env"
    trap 'rm -rf -- '"$(printf '%q' "$temp_dir")" EXIT
    backup_to_temp "$CONFIG_FILE" "$config_copy" || config_existed=false
    backup_to_temp "$ROUTER_ENV_FILE" "$env_copy" && env_existed=true
    service_running "$SERVICE_NAME" && was_running=true

    if ! python3 "$CONFIG_TOOL" --config "$CONFIG_FILE" --env "$ROUTER_ENV_FILE" "$@"; then
        restore_optional "$config_copy" "$CONFIG_FILE" "$config_existed"
        restore_optional "$env_copy" "$ROUTER_ENV_FILE" "$env_existed"
        die "No change was applied."
    fi

    if [[ "$was_running" == "true" ]]; then
        if ! systemctl restart "$SERVICE_NAME" || ! service_running "$SERVICE_NAME"; then
            restore_optional "$config_copy" "$CONFIG_FILE" "$config_existed"
            restore_optional "$env_copy" "$ROUTER_ENV_FILE" "$env_existed"
            systemctl restart "$SERVICE_NAME" 2>/dev/null || true
            die "The router could not start with that change, so the previous config was restored."
        fi
        echo "Saved and applied. Router restarted."
    else
        echo "Saved. Router is stopped; the change will apply when it starts."
    fi
    rm -rf -- "$temp_dir"
    trap - EXIT
}

endpoint_command() {
    local action="${1:-list}"
    shift || true
    need_config_tool
    case "$action" in
        list|ls) python3 "$CONFIG_TOOL" --config "$CONFIG_FILE" --env "$ROUTER_ENV_FILE" list ;;
        add|edit|remove|enable|disable) apply_router_mutation "$action" "$@" ;;
        help|-h|--help) show_endpoint_help ;;
        *) die "Unknown endpoint command: $action. Try: mla help endpoint" ;;
    esac
}

input_command() {
    local action="${1:-show}"
    shift || true
    case "$action" in
        show)
            need_config_tool
            python3 "$CONFIG_TOOL" --config "$CONFIG_FILE" --env "$ROUTER_ENV_FILE" list | awk 'NR == 1 || $4 == "input" || $1 == "input"'
            ;;
        uart) [[ $# -ge 1 ]] || die "Usage: sudo mla input uart DEVICE [BAUD]"; apply_router_mutation input-uart "$@" ;;
        udp) [[ $# -eq 2 ]] || die "Usage: sudo mla input udp ADDRESS PORT"; apply_router_mutation input-udp "$@" ;;
        help|-h|--help) show_input_help ;;
        *) die "Unknown input command: $action. Try: mla help input" ;;
    esac
}

raw_config_edit() {
    need_root
    need_config_tool
    [[ -f "$CONFIG_FILE" ]] || die "Router config not found: $CONFIG_FILE"
    local editor="${EDITOR:-nano}" temp_dir config_copy env_copy env_existed=false was_running=false
    command -v "${editor%% *}" >/dev/null || die "Editor not found: $editor"
    temp_dir="$(mktemp -d /tmp/mla-edit.XXXXXX)"
    config_copy="${temp_dir}/main.conf"
    env_copy="${temp_dir}/mavlink-router.env"
    trap 'rm -rf -- '"$(printf '%q' "$temp_dir")" EXIT
    cp -p "$CONFIG_FILE" "$config_copy"
    backup_to_temp "$ROUTER_ENV_FILE" "$env_copy" && env_existed=true
    service_running "$SERVICE_NAME" && was_running=true
    if ! $editor "$CONFIG_FILE"; then
        cp -p "$config_copy" "$CONFIG_FILE"
        restore_optional "$env_copy" "$ROUTER_ENV_FILE" "$env_existed"
        die "Editor exited with an error. The previous files were restored."
    fi

    if ! python3 "$CONFIG_TOOL" --config "$CONFIG_FILE" --env "$ROUTER_ENV_FILE" validate; then
        cp -p "$config_copy" "$CONFIG_FILE"
        die "Invalid config. The previous file was restored."
    fi
    if ! python3 "$CONFIG_TOOL" --config "$CONFIG_FILE" --env "$ROUTER_ENV_FILE" sync-env; then
        cp -p "$config_copy" "$CONFIG_FILE"
        restore_optional "$env_copy" "$ROUTER_ENV_FILE" "$env_existed"
        die "Could not update the companion environment file. The previous files were restored."
    fi
    if cmp -s "$config_copy" "$CONFIG_FILE"; then
        echo "No changes made."
        rm -rf -- "$temp_dir"
        trap - EXIT
        return 0
    fi
    local stamp
    stamp="$(date -u +%Y%m%dT%H%M%SZ)"
    cp -p "$config_copy" "${CONFIG_FILE}.backup.${stamp}"
    if [[ "$was_running" == "true" ]]; then
        if ! systemctl restart "$SERVICE_NAME" || ! service_running "$SERVICE_NAME"; then
            cp -p "$config_copy" "$CONFIG_FILE"
            restore_optional "$env_copy" "$ROUTER_ENV_FILE" "$env_existed"
            systemctl restart "$SERVICE_NAME" 2>/dev/null || true
            die "The router rejected the change. The previous files were restored."
        fi
        echo "Config saved and applied. Router restarted."
    else
        echo "Config saved. Router is stopped; it will apply on next start."
    fi
    rm -rf -- "$temp_dir"
    trap - EXIT
}

config_command() {
    local action="${1:-show}"
    case "$action" in
        show) [[ -f "$CONFIG_FILE" ]] || die "Router config not found: $CONFIG_FILE"; cat "$CONFIG_FILE" ;;
        path) printf 'Router config: %s\nRouter env:    %s\n' "$CONFIG_FILE" "$ROUTER_ENV_FILE" ;;
        check|validate) need_config_tool; python3 "$CONFIG_TOOL" --config "$CONFIG_FILE" --env "$ROUTER_ENV_FILE" validate ;;
        edit) raw_config_edit ;;
        help|-h|--help) show_config_help ;;
        *) die "Unknown config command: $action. Try: mla help config" ;;
    esac
}

show_status() {
    local router_state dashboard_state
    if service_running "$SERVICE_NAME"; then router_state="running"; else router_state="stopped"; fi
    if service_running "$DASHBOARD_SERVICE"; then dashboard_state="running"; else dashboard_state="stopped"; fi
    echo "MAVLink router:  $router_state"
    echo "Dashboard:       $dashboard_state ($(dashboard_current_listen))"
    echo "Config:          $CONFIG_FILE"
    echo
    if [[ -f "$CONFIG_FILE" && -f "$CONFIG_TOOL" ]]; then
        python3 "$CONFIG_TOOL" --config "$CONFIG_FILE" --env "$ROUTER_ENV_FILE" list || true
    else
        echo "No router configuration found."
    fi
    echo
    echo "More detail: systemctl status $SERVICE_NAME"
}

case "${1:-help}" in
    help|-h|--help) shift || true; show_help_topic "${1:-}" ;;
    version|--version) grep -m1 'MAVLINK_ANYWHERE_VERSION=' "${SCRIPT_DIR}/lib/common.sh" | cut -d'"' -f2 ;;
    status) show_status ;;
    logs|log) journalctl -u "$SERVICE_NAME" -f ;;
    restart) need_root; systemctl restart "$SERVICE_NAME"; echo "MAVLink router restarted." ;;
    stop) need_root; systemctl stop "$SERVICE_NAME"; echo "MAVLink router stopped." ;;
    start) need_root; systemctl start "$SERVICE_NAME"; echo "MAVLink router started." ;;
    endpoint|endpoints|route|routes|ep) shift; endpoint_command "$@" ;;
    input|source) shift; input_command "$@" ;;
    config|raw|edit) if [[ "$1" == "edit" ]]; then config_command edit; else shift; config_command "$@"; fi ;;
    dashboard|ui) shift; dashboard_command "$@" ;;
    reconfigure|reconfig|setup)
        [[ -x "${SCRIPT_DIR}/configure_mavlink_router.sh" ]] || die "Run configure from your Git checkout: cd ~/mavlink-anywhere && sudo ./configure_mavlink_router.sh"
        need_root
        exec "${SCRIPT_DIR}/configure_mavlink_router.sh"
        ;;
    *) die "Unknown command: $1. Try: mla help" ;;
esac
