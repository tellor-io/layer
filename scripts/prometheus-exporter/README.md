# Prometheus Price Data Exporter

A Go application that queries a Prometheus endpoint daily to collect price feed data and stores it in a PostgreSQL database. It also provides a password-protected REST API for accessing the stored data.

## Features

- Queries Prometheus API for `pricefeed_daemon_price_encoder_update_price` metrics
- Extracts price data with timestamps, market IDs, and exchange IDs
- Stores data in PostgreSQL with proper indexing
- Supports both one-time execution and daily scheduling
- **NEW**: Password-protected REST API for data access
- **NEW**: Multiple API endpoints for flexible data querying

## Database Schema

The application creates a `price_data` table with the following structure:

```sql
CREATE TABLE price_data (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    market_id VARCHAR(255) NOT NULL,
    exchange_id VARCHAR(255) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Configuration

The application uses environment variables for configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `PROMETHEUS_URL` | `http://54.160.217.166:9090` | Prometheus server URL |
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `postgres` | Database username |
| `DB_PASSWORD` | `password` | Database password |
| `DB_NAME` | `pricefeed` | Database name |
| `SCHEDULER_MODE` | `false` | Run in daily scheduler mode |
| `API_MODE` | `false` | Run as HTTP API server |
| `API_PASSWORD` | `admin123` | Password for API authentication |
| `API_PORT` | `8080` | Port for API server |

## Usage

### One-time execution

```bash
go run .
```

### Daily scheduler mode

```bash
SCHEDULER_MODE=true go run .
```

### API Server mode

```bash
API_MODE=true go run .
```

The API server will start on port 8080 (configurable via `API_PORT` environment variable).

## API Endpoints

All API endpoints require authentication via the `X-API-Key` header with the password set in `API_PASSWORD` environment variable.

### Authentication

All endpoints require the `X-API-Key` header:
```bash
curl -H "X-API-Key: admin123" http://localhost:8080/api/health
```

### Available Endpoints

#### Health Check
- **GET** `/api/health` - Returns server health status (no authentication required)

#### Price Data Endpoints
- **GET** `/api/prices` - Get all price data with optional filtering
- **GET** `/api/prices/latest` - Get latest price for each market/exchange combination
- **GET** `/api/prices/market/{market_id}` - Get prices for a specific market
- **GET** `/api/prices/exchange/{exchange_id}` - Get prices for a specific exchange
- **GET** `/api/prices/range` - Get prices within a date range

### Query Parameters

#### `/api/prices`
- `limit` - Number of records to return
- `offset` - Number of records to skip
- `market_id` - Filter by market ID
- `exchange_id` - Filter by exchange ID

#### `/api/prices/range`
- `start` - Start date (YYYY-MM-DD format)
- `end` - End date (YYYY-MM-DD format)
- `market_id` - Filter by market ID (optional)
- `exchange_id` - Filter by exchange ID (optional)
- `limit` - Number of records to return (optional)
- `offset` - Number of records to skip (optional)

### Example API Calls

```bash
# Get all prices (latest 10)
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices?limit=10"

# Get latest prices for all markets
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices/latest"

# Get prices for BTC-USD market
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices/market/BTC-USD"

# Get prices for Bitfinex exchange
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices/exchange/Bitfinex"

# Get prices for a date range
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices/range?start=2024-01-01&end=2024-01-31"

# Get prices for BTC-USD in a date range
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices/range?start=2024-01-01&end=2024-01-31&market_id=BTC-USD"
```

### Response Format

All API responses follow this format:
```json
{
  "success": true,
  "data": [...],
  "count": 100,
  "error": null
}
```

Error responses:
```json
{
  "success": false,
  "data": null,
  "error": "Error message"
}
```

## Data Collection

The application collects data from the previous day by default. It:

1. Calculates yesterday's date range (00:00:00 to 23:59:59 UTC)
2. Queries the Prometheus API with a 60-second step interval
3. Parses the JSON response to extract price data
4. Stores each data point in the database

## Sample Data

Based on the Prometheus response, the application extracts data like:

- **Timestamp**: Unix timestamp converted to readable format
- **Market ID**: e.g., "BTC-USD", "KING-USD"
- **Exchange ID**: e.g., "Bitfinex", "uniswapV4ethereum"
- **Price**: Decimal price value

## Error Handling

- Failed price parsing is logged but doesn't stop processing
- Database connection errors cause the application to exit
- HTTP errors are logged with response details
- Individual insert failures are logged but don't stop batch processing

## Development

### Prerequisites

- Go 1.21+
- PostgreSQL 12+

### Setup

1. Clone the repository
2. Install dependencies:
```bash
go mod download
```

3. Set up PostgreSQL database
4. Configure environment variables
5. Run the application

### Testing

Test with a specific date range by modifying the `collectAndStoreData` function to use custom start/end times.

## Monitoring

The application logs:
- Query URLs being executed
- Data collection progress
- Error conditions
- Successful completion

Monitor the logs to ensure daily data collection is working properly.
