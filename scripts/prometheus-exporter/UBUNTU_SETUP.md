# Ubuntu System Service Setup

This guide shows how to install the Prometheus Exporter as a systemd service on Ubuntu.

## Prerequisites

1. **Ubuntu 18.04+** with systemd
2. **PostgreSQL** installed and running
3. **Go 1.21+** installed
4. **Git** for cloning the repository

## Quick Setup

### 1. Install Prerequisites

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install PostgreSQL
sudo apt install postgresql postgresql-contrib -y

# Install Go (if not already installed)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Git
sudo apt install git -y
```

### 2. Setup PostgreSQL Database

```bash
# Switch to postgres user
sudo -u postgres psql

# Create database and user
CREATE DATABASE pricefeed;
CREATE USER prometheus_exporter WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE pricefeed TO prometheus_exporter;
\q
```

### 3. Install the Service

```bash
# Clone the repository (if not already done)
git clone <your-repo-url>
cd layer-repo/scripts/prometheus-exporter

# Make scripts executable
chmod +x install-service.sh manage-service.sh

# Install the service
sudo ./install-service.sh
```

### 4. Configure the Service

```bash
# Edit configuration
sudo nano /opt/prometheus-exporter/config.env
```

Update the configuration file with your settings:

```bash
# Prometheus Configuration
PROMETHEUS_URL=http://54.160.217.166:9090

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=prometheus_exporter
DB_PASSWORD=your_secure_password
DB_NAME=pricefeed

# API Configuration
API_PASSWORD=your_api_password
API_PORT=8080

# Application Mode
COMBINED_MODE=true
```

### 5. Start the Service

```bash
# Start the service
sudo systemctl start prometheus-exporter

# Enable auto-start on boot
sudo systemctl enable prometheus-exporter

# Check status
sudo systemctl status prometheus-exporter
```

## Service Management

Use the management script for easy control:

```bash
# Start service
./manage-service.sh start

# Stop service
./manage-service.sh stop

# Restart service
./manage-service.sh restart

# Check status
./manage-service.sh status

# View logs
./manage-service.sh logs

# View configuration
./manage-service.sh config

# Edit configuration
./manage-service.sh edit-config
```

## Manual Service Commands

If you prefer systemctl directly:

```bash
# Start service
sudo systemctl start prometheus-exporter

# Stop service
sudo systemctl stop prometheus-exporter

# Restart service
sudo systemctl restart prometheus-exporter

# Check status
sudo systemctl status prometheus-exporter

# View logs
sudo journalctl -u prometheus-exporter -f

# Enable auto-start
sudo systemctl enable prometheus-exporter

# Disable auto-start
sudo systemctl disable prometheus-exporter
```

## Testing the Service

### 1. Check Service Status

```bash
sudo systemctl status prometheus-exporter
```

### 2. Test API Endpoints

```bash
# Test health endpoint
curl http://localhost:8080/api/health

# Test API with authentication
curl -H "X-API-Key: your_api_password" "http://localhost:8080/api/prices/latest"
```

### 3. View Logs

```bash
# View recent logs
sudo journalctl -u prometheus-exporter --since "1 hour ago"

# Follow live logs
sudo journalctl -u prometheus-exporter -f
```

## Configuration

### Environment Variables

The service uses these environment variables (set in `/opt/prometheus-exporter/config.env`):

| Variable | Default | Description |
|----------|---------|-------------|
| `PROMETHEUS_URL` | `http://54.160.217.166:9090` | Prometheus server URL |
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `prometheus_exporter` | Database username |
| `DB_PASSWORD` | `password` | Database password |
| `DB_NAME` | `pricefeed` | Database name |
| `API_PASSWORD` | `admin123` | API authentication password |
| `API_PORT` | `8080` | API server port |
| `COMBINED_MODE` | `true` | Run API server + daily data collection |

### Security Features

The service includes these security features:

- Runs as dedicated `prometheus-exporter` user
- Restricted file system access
- No new privileges
- Private temporary directory
- Protected system directories

## Troubleshooting

### Service Won't Start

1. Check service status:
   ```bash
   sudo systemctl status prometheus-exporter
   ```

2. Check logs for errors:
   ```bash
   sudo journalctl -u prometheus-exporter --since "10 minutes ago"
   ```

3. Verify database connection:
   ```bash
   sudo -u prometheus-exporter psql -h localhost -U prometheus_exporter -d pricefeed
   ```

### API Not Responding

1. Check if service is running:
   ```bash
   sudo systemctl status prometheus-exporter
   ```

2. Check if port is listening:
   ```bash
   sudo netstat -tlnp | grep 8080
   ```

3. Check firewall settings:
   ```bash
   sudo ufw status
   sudo ufw allow 8080
   ```

### Database Connection Issues

1. Test database connection:
   ```bash
   sudo -u prometheus-exporter psql -h localhost -U prometheus_exporter -d pricefeed
   ```

2. Check PostgreSQL status:
   ```bash
   sudo systemctl status postgresql
   ```

3. Verify database exists:
   ```bash
   sudo -u postgres psql -c "\l"
   ```

## Uninstalling

To completely remove the service:

```bash
# Using management script
./manage-service.sh uninstall

# Or manually
sudo systemctl stop prometheus-exporter
sudo systemctl disable prometheus-exporter
sudo rm /etc/systemd/system/prometheus-exporter.service
sudo userdel prometheus-exporter
sudo rm -rf /opt/prometheus-exporter
sudo systemctl daemon-reload
```

## Monitoring

### Log Rotation

Create log rotation configuration:

```bash
sudo nano /etc/logrotate.d/prometheus-exporter
```

Add:
```
/var/log/prometheus-exporter/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 644 prometheus-exporter prometheus-exporter
}
```

### Health Monitoring

Create a simple health check script:

```bash
#!/bin/bash
# /opt/prometheus-exporter/health-check.sh

API_URL="http://localhost:8080/api/health"
if curl -f -s "$API_URL" > /dev/null; then
    echo "Service is healthy"
    exit 0
else
    echo "Service is unhealthy"
    exit 1
fi
```

## Production Considerations

1. **Change default passwords** in the configuration file
2. **Set up SSL/TLS** for the API if needed
3. **Configure firewall** rules for the API port
4. **Set up log monitoring** and alerting
5. **Regular database backups**
6. **Monitor disk space** for database growth
7. **Set up monitoring** for the service health
