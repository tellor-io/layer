#!/bin/bash

# Management script for Prometheus Exporter systemd service
# Usage: ./manage-service.sh [start|stop|restart|status|logs|config|uninstall]

SERVICE_NAME="prometheus-exporter"
INSTALL_DIR="/opt/prometheus-exporter"
CONFIG_FILE="$INSTALL_DIR/config.env"

case "$1" in
    start)
        echo "Starting $SERVICE_NAME service..."
        sudo systemctl start "$SERVICE_NAME"
        echo "Service started"
        ;;
    stop)
        echo "Stopping $SERVICE_NAME service..."
        sudo systemctl stop "$SERVICE_NAME"
        echo "Service stopped"
        ;;
    restart)
        echo "Restarting $SERVICE_NAME service..."
        sudo systemctl restart "$SERVICE_NAME"
        echo "Service restarted"
        ;;
    status)
        echo "Checking $SERVICE_NAME service status..."
        sudo systemctl status "$SERVICE_NAME"
        ;;
    logs)
        echo "Showing $SERVICE_NAME service logs (press Ctrl+C to exit)..."
        sudo journalctl -u "$SERVICE_NAME" -f
        ;;
    config)
        echo "Current configuration:"
        if [ -f "$CONFIG_FILE" ]; then
            cat "$CONFIG_FILE"
        else
            echo "Configuration file not found at $CONFIG_FILE"
        fi
        ;;
    edit-config)
        echo "Editing configuration file..."
        if [ -f "$CONFIG_FILE" ]; then
            sudo nano "$CONFIG_FILE"
            echo "Configuration updated. Restart the service to apply changes:"
            echo "  sudo systemctl restart $SERVICE_NAME"
        else
            echo "Configuration file not found at $CONFIG_FILE"
        fi
        ;;
    uninstall)
        echo "Uninstalling $SERVICE_NAME service..."
        sudo systemctl stop "$SERVICE_NAME" 2>/dev/null || true
        sudo systemctl disable "$SERVICE_NAME" 2>/dev/null || true
        sudo rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
        sudo userdel "$SERVICE_NAME" 2>/dev/null || true
        sudo rm -rf "$INSTALL_DIR"
        sudo systemctl daemon-reload
        echo "Service uninstalled"
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs|config|edit-config|uninstall}"
        echo ""
        echo "Commands:"
        echo "  start       - Start the service"
        echo "  stop        - Stop the service"
        echo "  restart     - Restart the service"
        echo "  status      - Show service status"
        echo "  logs        - Show live logs"
        echo "  config      - Show current configuration"
        echo "  edit-config - Edit configuration file"
        echo "  uninstall   - Remove the service completely"
        exit 1
        ;;
esac
