#!/bin/bash

# Test script for CSV export functionality
# Make sure the API server is running before executing this script

API_BASE_URL=""
API_KEY="admin123"  # Default API key, change if different

echo "Testing CSV Export Functionality"
echo "==============================="

echo ""
echo "1. Testing /api/prices with CSV format:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices?format=csv&limit=5" -o prices.csv
echo "Downloaded prices.csv (first 5 records)"

echo ""
echo "2. Testing /api/prices/latest with CSV format:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/latest?format=csv" -o latest_prices.csv
echo "Downloaded latest_prices.csv"

echo ""
echo "3. Testing /api/prices/market/BTC-USD with CSV format:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/market/BTC-USD?format=csv&limit=10" -o btc_prices.csv
echo "Downloaded btc_prices.csv (BTC-USD market, first 10 records)"

echo ""
echo "4. Testing /api/prices/exchange/binance with CSV format:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/exchange/binance?format=csv&limit=10" -o binance_prices.csv
echo "Downloaded binance_prices.csv (Binance exchange, first 10 records)"

echo ""
echo "5. Testing /api/prices/range with CSV format:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/range?start=2024-01-01&end=2024-01-31&format=csv&limit=20" -o january_prices.csv
echo "Downloaded january_prices.csv (January 2024 data, first 20 records)"

echo ""
echo "6. Testing JSON format (for comparison):"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices?limit=3" | jq '.'

echo ""
echo "CSV Export Test Complete!"
echo "Check the generated CSV files for the exported data."
echo ""
echo "Generated files:"
ls -la *.csv
