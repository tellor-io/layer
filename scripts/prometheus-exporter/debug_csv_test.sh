#!/bin/bash

# Debug script to test CSV export functionality
API_BASE_URL="http://18.208.245.152:8080"
API_KEY="admin123"

echo "Testing CSV export with different parameter formats..."
echo "=================================================="

echo ""
echo "1. Testing with format=csv:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/latest?format=csv" -o test1.csv
echo "Response saved to test1.csv"
echo "First few lines of test1.csv:"
head -3 test1.csv

echo ""
echo "2. Testing with format=CSV (uppercase):"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/latest?format=CSV" -o test2.csv
echo "Response saved to test2.csv"
echo "First few lines of test2.csv:"
head -3 test2.csv

echo ""
echo "3. Testing with format=csv&limit=5:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/latest?format=csv&limit=5" -o test3.csv
echo "Response saved to test3.csv"
echo "First few lines of test3.csv:"
head -3 test3.csv

echo ""
echo "4. Testing JSON format for comparison:"
curl -H "X-API-Key: $API_KEY" "$API_BASE_URL/api/prices/latest?limit=3" -o test4.json
echo "Response saved to test4.json"
echo "First few lines of test4.json:"
head -3 test4.json

echo ""
echo "5. Checking file types:"
file test1.csv test2.csv test3.csv test4.json

echo ""
echo "6. Checking if any CSV files contain actual CSV data:"
for file in test1.csv test2.csv test3.csv; do
    echo "=== $file ==="
    if grep -q "timestamp,market_id,exchange_id,price" "$file"; then
        echo "✓ Contains CSV header"
    else
        echo "✗ Missing CSV header"
    fi
    if grep -q "{" "$file"; then
        echo "✗ Contains JSON data"
    else
        echo "✓ No JSON data detected"
    fi
done
