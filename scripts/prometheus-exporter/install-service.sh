#!/bin/bash

# Installation script for Prometheus Exporter systemd service
# Run this script with sudo

set -e

echo "Installing Prometheus Exporter as a systemd service..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run this script with sudo"
    exit 1
fi

# Configuration
SERVICE_NAME="prometheus-exporter"
SERVICE_USER="prometheus-exporter"
INSTALL_DIR="/opt/prometheus-exporter"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Installing to: $INSTALL_DIR"

# Create service user
echo "Creating service user: $SERVICE_USER"
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
    echo "User $SERVICE_USER created"
else
    echo "User $SERVICE_USER already exists"
fi

# Create installation directory
echo "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"

# Build the Go application
echo "Building Go application..."
cd "$CURRENT_DIR"
go build -o "$INSTALL_DIR/prometheus-exporter" .

# Set proper permissions
chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/prometheus-exporter"
chmod +x "$INSTALL_DIR/prometheus-exporter"

# Copy service file
echo "Installing systemd service file..."
cp "$CURRENT_DIR/prometheus-exporter.service" "$SERVICE_FILE"

# Copy config file
echo "Installing configuration file..."
cp "$CURRENT_DIR/config.env" "$INSTALL_DIR/config.env"
chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/config.env"
chmod 600 "$INSTALL_DIR/config.env"

# Reload systemd
echo "Reloading systemd daemon..."
systemctl daemon-reload

# Enable service
echo "Enabling service to start on boot..."
systemctl enable "$SERVICE_NAME"

echo ""
echo "Installation complete!"
echo ""
echo "To start the service:"
echo "  sudo systemctl start $SERVICE_NAME"
echo ""
echo "To check status:"
echo "  sudo systemctl status $SERVICE_NAME"
echo ""
echo "To view logs:"
echo "  sudo journalctl -u $SERVICE_NAME -f"
echo ""
echo "To stop the service:"
echo "  sudo systemctl stop $SERVICE_NAME"
echo ""
echo "To uninstall:"
echo "  sudo systemctl stop $SERVICE_NAME"
echo "  sudo systemctl disable $SERVICE_NAME"
echo "  sudo rm $SERVICE_FILE"
echo "  sudo userdel $SERVICE_USER"
echo "  sudo rm -rf $INSTALL_DIR"
