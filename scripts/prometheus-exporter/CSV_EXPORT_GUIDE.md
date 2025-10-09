# CSV Export Guide

The Prometheus Exporter API now supports CSV export functionality for all price data endpoints. This allows you to easily download data in CSV format for analysis in spreadsheet applications or other data processing tools.

## How to Use CSV Export

Add the query parameter `format=csv` to any of the existing API endpoints to get CSV data instead of JSON.

### Available Endpoints with CSV Support

1. **Get All Prices**: `/api/prices?format=csv`
2. **Get Latest Prices**: `/api/prices/latest?format=csv`
3. **Get Prices by Market**: `/api/prices/market/{market_id}?format=csv`
4. **Get Prices by Exchange**: `/api/prices/exchange/{exchange_id}?format=csv`
5. **Get Prices by Date Range**: `/api/prices/range?start=YYYY-MM-DD&end=YYYY-MM-DD&format=csv`

### CSV Format

The CSV output includes the following columns:
- `timestamp`: Date and time in YYYY-MM-DD HH:MM:SS format
- `market_id`: Market identifier (e.g., BTC-USD)
- `exchange_id`: Exchange identifier (e.g., binance)
- `price`: Price value with 8 decimal places

### Example Usage

#### 1. Download all prices as CSV:
```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8080/api/prices?format=csv" -o all_prices.csv
```

#### 2. Download latest prices as CSV:
```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8080/api/prices/latest?format=csv" -o latest_prices.csv
```

#### 3. Download BTC-USD prices as CSV:
```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8080/api/prices/market/BTC-USD?format=csv" -o btc_prices.csv
```

#### 4. Download Binance exchange prices as CSV:
```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8080/api/prices/exchange/binance?format=csv" -o binance_prices.csv
```

#### 5. Download January 2024 prices as CSV:
```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8080/api/prices/range?start=2024-01-01&end=2024-01-31&format=csv" -o january_prices.csv
```

### Query Parameters

All existing query parameters work with CSV export:

- `limit`: Limit the number of records returned
- `offset`: Skip a number of records
- `market_id`: Filter by market ID (for /api/prices endpoint)
- `exchange_id`: Filter by exchange ID (for /api/prices endpoint)
- `start` and `end`: Date range (for /api/prices/range endpoint)

### Example with Parameters:
```bash
curl -H "X-API-Key: your-api-key" "http://localhost:8080/api/prices?format=csv&limit=100&market_id=BTC-USD" -o btc_recent.csv
```

### Error Handling

If an error occurs when requesting CSV format, the response will be a simple CSV with an error message:
```
error,Error description here
```

### File Download

When you request CSV format, the server sets appropriate headers to trigger a file download in your browser:
- `Content-Type: text/csv`
- `Content-Disposition: attachment; filename=price_data.csv`

### Testing

Use the provided test script to verify CSV export functionality:
```bash
./test_csv_export.sh
```

This will test all endpoints and download sample CSV files.

### Integration with Analysis Tools

The CSV format is compatible with:
- Microsoft Excel
- Google Sheets
- Python pandas
- R data frames
- Tableau
- Power BI
- Any other tool that can read CSV files

### Performance Considerations

- CSV export is more efficient than JSON for large datasets
- Use `limit` parameter to control the amount of data downloaded
- Consider using date ranges to limit data to specific time periods
- The CSV format reduces response size compared to JSON for large datasets
