#!/bin/bash
# =============================================================================
# MAVLink-Anywhere: CLI Helper
# =============================================================================
# Quick commands for managing mavlink-router
# Usage: ./mavlink-router-cli.sh [command]
# =============================================================================

SERVICE_NAME="mavlink-router"
CONFIG_FILE="/etc/mavlink-router/main.conf"
ENV_FILE="/etc/default/mavlink-router"

show_help() {
    echo "MAVLink-Anywhere CLI Helper"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  status      Show service status and current config"
    echo "  logs        Show live logs (Ctrl+C to exit)"
    echo "  restart     Restart the service"
    echo "  stop        Stop the service"
    echo "  start       Start the service"
    echo "  config      Show current configuration"
    echo "  edit        Edit configuration file"
    echo "  endpoints   Quick edit UDP endpoints"
    echo "  reconfigure Run the configuration wizard again"
    echo "  help        Show this help"
    echo ""
}

show_status() {
    echo "=== MAVLink Router Status ==="
    systemctl status $SERVICE_NAME --no-pager 2>/dev/null || echo "Service not installed"
    echo ""
    echo "=== Current Configuration ==="
    if [[ -f "$CONFIG_FILE" ]]; then
        cat "$CONFIG_FILE"
    else
        echo "No configuration file found"
    fi
}

show_logs() {
    echo "Showing live logs (Ctrl+C to exit)..."
    journalctl -u $SERVICE_NAME -f
}

show_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        echo "=== Configuration File: $CONFIG_FILE ==="
        cat "$CONFIG_FILE"
    else
        echo "No configuration file found at $CONFIG_FILE"
    fi
    echo ""
    if [[ -f "$ENV_FILE" ]]; then
        echo "=== Environment File: $ENV_FILE ==="
        cat "$ENV_FILE"
    fi
}

edit_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        ${EDITOR:-nano} "$CONFIG_FILE"
        echo ""
        read -p "Restart service to apply changes? [Y/n]: " RESTART
        if [[ "${RESTART,,}" != "n" ]]; then
            sudo systemctl restart $SERVICE_NAME
            echo "Service restarted"
        fi
    else
        echo "No configuration file found. Run configure_mavlink_router.sh first."
    fi
}

edit_endpoints() {
    if [[ ! -f "$ENV_FILE" ]]; then
        echo "No configuration found. Run configure_mavlink_router.sh first."
        exit 1
    fi

    # Load current settings
    source "$ENV_FILE"

    echo "Current UDP endpoints: ${UDP_ENDPOINTS}"
    echo ""
    echo "Enter new endpoints (space-separated, e.g., '127.0.0.1:14550 192.168.1.100:14550'):"
    read -p "New endpoints: " NEW_ENDPOINTS

    if [[ -z "$NEW_ENDPOINTS" ]]; then
        echo "No changes made"
        exit 0
    fi

    # Update environment file
    sudo sed -i "s|^UDP_ENDPOINTS=.*|UDP_ENDPOINTS=\"${NEW_ENDPOINTS}\"|" "$ENV_FILE"

    # Regenerate config file
    echo "Regenerating configuration..."

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "${SCRIPT_DIR}/configure_mavlink_router.sh" ]]; then
        # Re-run configuration with current settings
        sudo "${SCRIPT_DIR}/configure_mavlink_router.sh" --headless \
            --uart "${UART_DEVICE:-/dev/ttyS0}" \
            --baud "${UART_BAUD:-57600}" \
            --endpoints "${NEW_ENDPOINTS// /,}"
    else
        echo "Configuration script not found. Please edit $CONFIG_FILE manually."
    fi
}

reconfigure() {
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "${SCRIPT_DIR}/configure_mavlink_router.sh" ]]; then
        sudo "${SCRIPT_DIR}/configure_mavlink_router.sh"
    else
        echo "Configuration script not found"
    fi
}

# Main
case "${1:-help}" in
    status)
        show_status
        ;;
    logs|log)
        show_logs
        ;;
    restart)
        sudo systemctl restart $SERVICE_NAME
        echo "Service restarted"
        systemctl status $SERVICE_NAME --no-pager
        ;;
    stop)
        sudo systemctl stop $SERVICE_NAME
        echo "Service stopped"
        ;;
    start)
        sudo systemctl start $SERVICE_NAME
        echo "Service started"
        systemctl status $SERVICE_NAME --no-pager
        ;;
    config|show)
        show_config
        ;;
    edit)
        edit_config
        ;;
    endpoints|ep)
        edit_endpoints
        ;;
    reconfigure|reconfig)
        reconfigure
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
