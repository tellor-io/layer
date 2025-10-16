#!/bin/bash

# Test script for the Prometheus Exporter API
# Make sure to set your API_PASSWORD environment variable or use the default

API_URL="http://localhost:8080"
API_KEY="${API_PASSWORD:-admin123}"

echo "Testing Prometheus Exporter API..."
echo "API URL: $API_URL"
echo "API Key: $API_KEY"
echo ""

# Test health endpoint (no auth required)
echo "1. Testing health endpoint..."
curl -s "$API_URL/api/health" | jq .
echo ""

# Test prices endpoint with auth
echo "2. Testing prices endpoint with authentication..."
curl -s -H "X-API-Key: $API_KEY" "$API_URL/api/prices?limit=5" | jq .
echo ""

# Test latest prices endpoint
echo "3. Testing latest prices endpoint..."
curl -s -H "X-API-Key: $API_KEY" "$API_URL/api/prices/latest" | jq .
echo ""

# Test market-specific endpoint
echo "4. Testing market-specific endpoint..."
curl -s -H "X-API-Key: $API_KEY" "$API_URL/api/prices/market/BTC-USD" | jq .
echo ""

# Test exchange-specific endpoint
echo "5. Testing exchange-specific endpoint..."
curl -s -H "X-API-Key: $API_KEY" "$API_URL/api/prices/exchange/Bitfinex" | jq .
echo ""

# Test date range endpoint
echo "6. Testing date range endpoint..."
curl -s -H "X-API-Key: $API_KEY" "$API_URL/api/prices/range?start=2024-01-01&end=2024-01-31" | jq .
echo ""

# Test authentication failure
echo "7. Testing authentication failure..."
curl -s -H "X-API-Key: wrongpassword" "$API_URL/api/prices" | jq .
echo ""

echo "API testing completed!"
