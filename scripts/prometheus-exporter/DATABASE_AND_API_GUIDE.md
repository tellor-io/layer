# Database Connection and API Usage Guide

This guide provides instructions for connecting to the Prometheus Exporter database and using the API to download data.

## Table of Contents
1. [Database Connection](#database-connection)
   - [Terminal (psql)](#terminal-psql)
   - [Python](#python)
   - [Golang](#golang)
2. [API Usage](#api-usage)
   - [Authentication](#authentication)
   - [Available Endpoints](#available-endpoints)
   - [JSON Export](#json-export)
   - [CSV Export](#csv-export)

---

## Database Connection

### Default Configuration

- **Host**: `localhost`
- **Port**: `5432`
- **User**: `postgres`
- **Password**: `password`
- **Database**: `pricefeed`

### Database Schema

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

---

## Terminal (psql)

### Installation

**macOS:** `brew install postgresql`  
**Ubuntu/Debian:** `sudo apt-get install postgresql-client`  
**RHEL/CentOS:** `sudo yum install postgresql`

### Connect to Database

```bash
# Using psql
psql -h localhost -p 5432 -U postgres -d pricefeed

# Using connection string
psql postgresql://postgres:password@localhost:5432/pricefeed

# With password in environment
PGPASSWORD=password psql -h localhost -p 5432 -U postgres -d pricefeed
```

### Common Queries

```sql
-- View all tables
\dt

-- Get latest prices
SELECT * FROM price_data ORDER BY timestamp DESC LIMIT 10;

-- Filter by market
SELECT * FROM price_data WHERE market_id = 'BTC-USD' ORDER BY timestamp DESC;

-- Filter by exchange
SELECT * FROM price_data WHERE exchange_id = 'binance' ORDER BY timestamp DESC;

-- Date range query
SELECT * FROM price_data 
WHERE timestamp BETWEEN '2024-01-01' AND '2024-01-31' 
ORDER BY timestamp DESC;

-- Aggregations
SELECT market_id, AVG(price) as avg_price, COUNT(*) as count 
FROM price_data 
GROUP BY market_id;
```

### Export to CSV from Terminal

```bash
psql postgresql://postgres:password@localhost:5432/pricefeed \
  -c "COPY (SELECT * FROM price_data LIMIT 1000) TO STDOUT WITH CSV HEADER" > export.csv
```

---

## Python

### Installation

```bash
pip install psycopg2-binary
```

### Basic Connection

```python
import psycopg2

# Connect to database
conn = psycopg2.connect(
    host="localhost",
    port=5432,
    user="postgres",
    password="password",
    database="pricefeed"
)

# Create cursor and execute query
cursor = conn.cursor()
cursor.execute("SELECT * FROM price_data LIMIT 10")
rows = cursor.fetchall()

# Process results
for row in rows:
    print(row)

# Clean up
cursor.close()
conn.close()
```

### Query with Parameters

```python
# Safe parameterized query
cursor.execute(
    "SELECT * FROM price_data WHERE market_id = %s LIMIT %s",
    ("BTC-USD", 100)
)
results = cursor.fetchall()
```

### Using Pandas

```python
import pandas as pd
import psycopg2

conn = psycopg2.connect("postgresql://postgres:password@localhost:5432/pricefeed")
df = pd.read_sql_query("SELECT * FROM price_data LIMIT 1000", conn)
conn.close()

# Now you can analyze with pandas
print(df.describe())
print(df.groupby('market_id')['price'].mean())
```

---

## Golang

### Installation

```bash
go get github.com/lib/pq
```

### Basic Connection

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    _ "github.com/lib/pq"
)

func main() {
    // Connect to database
    connStr := "host=localhost port=5432 user=postgres password=password dbname=pricefeed sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Test connection
    if err := db.Ping(); err != nil {
        log.Fatal(err)
    }

    // Query data
    rows, err := db.Query("SELECT timestamp, market_id, exchange_id, price FROM price_data LIMIT 10")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    // Process results
    for rows.Next() {
        var timestamp time.Time
        var marketID, exchangeID string
        var price float64
        
        rows.Scan(&timestamp, &marketID, &exchangeID, &price)
        fmt.Printf("%s - %s/%s: $%.2f\n", timestamp, marketID, exchangeID, price)
    }
}
```

### Query with Parameters

```go
// Parameterized query
rows, err := db.Query(
    "SELECT * FROM price_data WHERE market_id = $1 LIMIT $2",
    "BTC-USD", 100
)
```

---

## API Usage

### Authentication

All API endpoints (except `/api/health`) require authentication using an API key in the `X-API-Key` header.

**Default API Key:** `admin123` (configurable via `API_PASSWORD` environment variable)

**Example:**
```bash
curl -H "X-API-Key: admin123" http://localhost:8080/api/prices
```

### Base URL

Default: `http://localhost:8080`

---

## Available Endpoints

### 1. Health Check (No Auth)
```bash
curl http://localhost:8080/api/health
```

### 2. Get All Prices
```bash
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices"
```
**Query Parameters:** `limit`, `offset`, `market_id`, `exchange_id`

### 3. Get Latest Prices
Get the most recent price for each market/exchange pair.
```bash
curl -H "X-API-Key: admin123" http://localhost:8080/api/prices/latest
```

### 4. Get Prices by Market
```bash
curl -H "X-API-Key: admin123" http://localhost:8080/api/prices/market/BTC-USD
```
**Query Parameters:** `limit`, `offset`

### 5. Get Prices by Exchange
```bash
curl -H "X-API-Key: admin123" http://localhost:8080/api/prices/exchange/binance
```
**Query Parameters:** `limit`, `offset`

### 6. Get Prices by Date Range
```bash
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices/range?start=2024-01-01&end=2024-01-31"
```
**Query Parameters (Required):** `start`, `end` (format: YYYY-MM-DD)  
**Query Parameters (Optional):** `market_id`, `exchange_id`, `limit`, `offset`

---

## JSON Export

### Response Format

```json
{
  "success": true,
  "data": [
    {
      "Timestamp": "2024-01-15T12:30:00Z",
      "MarketID": "BTC-USD",
      "ExchangeID": "binance",
      "Price": 42500.12345678
    }
  ],
  "count": 1
}
```

### Using JSON API in Code

**Python:**
```python
import requests

response = requests.get(
    "http://localhost:8080/api/prices/latest",
    headers={"X-API-Key": "admin123"}
)
data = response.json()
print(f"Retrieved {data['count']} prices")
```

**JavaScript:**
```javascript
const response = await fetch('http://localhost:8080/api/prices/latest', {
    headers: { 'X-API-Key': 'admin123' }
});
const data = await response.json();
console.log(`Retrieved ${data.count} prices`);
```

**Golang:**
```go
req, _ := http.NewRequest("GET", "http://localhost:8080/api/prices/latest", nil)
req.Header.Set("X-API-Key", "admin123")
resp, _ := http.DefaultClient.Do(req)
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
// Parse JSON from body
```

---

## CSV Export

### How to Use

Add `?format=csv` to any endpoint to download data as CSV instead of JSON.

### CSV Format

Columns: `timestamp`, `market_id`, `exchange_id`, `price`

### Examples

**Download Latest Prices:**
```bash
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices/latest?format=csv" \
  -o latest_prices.csv
```

**Download Market Data:**
```bash
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices/market/BTC-USD?format=csv&limit=1000" \
  -o btc_prices.csv
```

**Download Date Range:**
```bash
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices/range?start=2024-01-01&end=2024-01-31&format=csv" \
  -o january_2024.csv
```

**Download with Filters:**
```bash
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices?market_id=BTC-USD&exchange_id=binance&format=csv&limit=500" \
  -o btc_binance.csv
```

### Using CSV in Python

```python
import requests
import pandas as pd
from io import StringIO

response = requests.get(
    "http://localhost:8080/api/prices/latest?format=csv",
    headers={"X-API-Key": "admin123"}
)

# Load into pandas DataFrame
df = pd.read_csv(StringIO(response.text))
print(df.head())
```

---

## Query Parameters Reference

### Common Parameters (Most Endpoints)
- `format=csv` - Return CSV instead of JSON
- `limit=N` - Limit number of results
- `offset=N` - Skip N records (pagination)

### Filtering Parameters
- `market_id=X` - Filter by market (e.g., BTC-USD)
- `exchange_id=X` - Filter by exchange (e.g., binance)

### Date Range Parameters (`/api/prices/range`)
- `start=YYYY-MM-DD` - Start date (required)
- `end=YYYY-MM-DD` - End date (required)

---

## Error Handling

### JSON Error Response
```json
{
  "success": false,
  "error": "Error message here"
}
```

### HTTP Status Codes
- `200` - Success
- `400` - Bad Request (invalid parameters)
- `401` - Unauthorized (invalid/missing API key)
- `500` - Internal Server Error

---

## Best Practices

### Database
- Always close connections when done
- Use parameterized queries to prevent SQL injection
- Use connection pooling for production applications
- Filter data at the database level rather than in application code

### API
- Store API keys securely (use environment variables)
- Use pagination (`limit`/`offset`) for large datasets
- Use CSV format for large data exports (more efficient)
- Filter at the API level (market_id, exchange_id, date range)
- Implement retry logic for transient failures
- Set reasonable timeouts for HTTP requests

---

## Troubleshooting

### Cannot Connect to Database
```bash
# Test connection
psql postgresql://postgres:password@localhost:5432/pricefeed -c "SELECT 1"
```

### API Returns 401 Error
Check your API key matches the configured password:
```bash
grep API_PASSWORD config.env
```

### Empty Results
Verify data exists:
```sql
SELECT COUNT(*) FROM price_data;
SELECT market_id, COUNT(*) FROM price_data GROUP BY market_id;
```

---

## Additional Resources

- PostgreSQL Documentation: https://www.postgresql.org/docs/
- Python psycopg2: https://www.psycopg.org/docs/
- Golang pq driver: https://github.com/lib/pq

For quick reference, see [QUICK_REFERENCE.md](./QUICK_REFERENCE.md)
