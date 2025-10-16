# Quick Reference Guide

Quick reference for common database and API operations.

---

## Database Connection

**Terminal:**
```bash
psql postgresql://postgres:password@localhost:5432/pricefeed
```

**Python:**
```python
import psycopg2
conn = psycopg2.connect("postgresql://postgres:password@localhost:5432/pricefeed")
```

**Golang:**
```go
db, _ := sql.Open("postgres", "host=localhost port=5432 user=postgres password=password dbname=pricefeed sslmode=disable")
```

---

## Common SQL Queries

```sql
-- Latest prices
SELECT * FROM price_data ORDER BY timestamp DESC LIMIT 10;

-- By market
SELECT * FROM price_data WHERE market_id = 'BTC-USD' LIMIT 100;

-- By exchange
SELECT * FROM price_data WHERE exchange_id = 'binance' LIMIT 100;

-- Date range
SELECT * FROM price_data WHERE timestamp BETWEEN '2024-01-01' AND '2024-01-31';

-- Aggregations
SELECT market_id, AVG(price) FROM price_data GROUP BY market_id;
```

---

## API Endpoints

**Base URL:** `http://localhost:8080`  
**Authentication:** Add header `X-API-Key: admin123`

```bash
# Health check (no auth)
curl http://localhost:8080/api/health

# Latest prices
curl -H "X-API-Key: admin123" http://localhost:8080/api/prices/latest

# All prices
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices?limit=100"

# By market
curl -H "X-API-Key: admin123" http://localhost:8080/api/prices/market/BTC-USD

# By exchange
curl -H "X-API-Key: admin123" http://localhost:8080/api/prices/exchange/binance

# Date range
curl -H "X-API-Key: admin123" "http://localhost:8080/api/prices/range?start=2024-01-01&end=2024-01-31"
```

---

## CSV Export

Add `?format=csv` to any endpoint:

```bash
# Latest prices as CSV
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices/latest?format=csv" -o latest.csv

# Market data as CSV
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices/market/BTC-USD?format=csv" -o btc.csv

# Date range as CSV
curl -H "X-API-Key: admin123" \
  "http://localhost:8080/api/prices/range?start=2024-01-01&end=2024-01-31&format=csv" -o data.csv
```

---

## Query Parameters

**All Endpoints:**
- `format=csv` - Return CSV instead of JSON
- `limit=N` - Limit results
- `offset=N` - Skip records (pagination)

**Filter Parameters:**
- `market_id=X` - Filter by market
- `exchange_id=X` - Filter by exchange

**Date Range (`/api/prices/range`):**
- `start=YYYY-MM-DD` - Start date (required)
- `end=YYYY-MM-DD` - End date (required)

---

## Configuration

Default values (set in `config.env`):

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=pricefeed

API_PASSWORD=admin123
API_PORT=8080
```

---

## Troubleshooting

**Test database connection:**
```bash
psql postgresql://postgres:password@localhost:5432/pricefeed -c "SELECT 1"
```

**Test API:**
```bash
curl http://localhost:8080/api/health
```

**Check data count:**
```sql
SELECT COUNT(*) FROM price_data;
```

---

For detailed documentation, see [DATABASE_AND_API_GUIDE.md](./DATABASE_AND_API_GUIDE.md)
