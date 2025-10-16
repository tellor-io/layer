#!/bin/bash

# Startup script for Prometheus Exporter
# This script sets up environment variables and starts the application

echo "Starting Prometheus Exporter..."

# Set default environment variables if not already set
export PROMETHEUS_URL="${PROMETHEUS_URL:-http://54.160.217.166:9090}"
export DB_HOST="${DB_HOST:-localhost}"
export DB_PORT="${DB_PORT:-5432}"
export DB_USER="${DB_USER:-postgres}"
export DB_PASSWORD="${DB_PASSWORD:-password}"
export DB_NAME="${DB_NAME:-pricefeed}"
export API_PASSWORD="${API_PASSWORD:-admin123}"
export API_PORT="${API_PORT:-8080}"

# Check if COMBINED_MODE is set, otherwise default to true
if [ -z "$COMBINED_MODE" ] && [ -z "$API_MODE" ] && [ -z "$SCHEDULER_MODE" ]; then
    export COMBINED_MODE="true"
    echo "No mode specified, defaulting to COMBINED_MODE=true"
fi

echo "Configuration:"
echo "  Prometheus URL: $PROMETHEUS_URL"
echo "  Database: $DB_USER@$DB_HOST:$DB_PORT/$DB_NAME"
echo "  API Port: $API_PORT"
echo "  API Password: $API_PASSWORD"
echo ""

if [ "$COMBINED_MODE" = "true" ]; then
    echo "Starting in COMBINED mode (API server + daily data collection)..."
    COMBINED_MODE=true go run .
elif [ "$API_MODE" = "true" ]; then
    echo "Starting in API mode..."
    API_MODE=true go run .
elif [ "$SCHEDULER_MODE" = "true" ]; then
    echo "Starting in SCHEDULER mode..."
    SCHEDULER_MODE=true go run .
else
    echo "Starting one-time data collection..."
    go run .
fi
