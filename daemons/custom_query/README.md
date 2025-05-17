# Custom Query Service

## Overview

The Custom Query Service allows you to fetch data from multiple API sources, aggregate the results, and prepare them for Layer reporting. This system is designed to handle data queries not covered by [pricefeed client](../pricefeed/).

## How It Works

1. You define API endpoints in the config file
2. You create queries that use these endpoints
3. The service fetches data from all endpoints for a query
4. Results are aggregated (using median or other methods)
5. Data is encoded according to the specified response type

## Configuration

All configuration is done through the `config.toml` file with two main sections:

### 1. Endpoint Definitions

Each endpoint represents an API source:

```toml
[endpoints.coingecko]
url_template = "https://api.coingecko.com/api/v3/simple/price?ids={coin_id}&vs_currencies=usd"
method = "GET"
timeout = 5000
```

Endpoints support:
- URL templates with placeholders (like `{coin_id}`)
- HTTP method specification
- Custom timeouts
- API keys via environment variables (`${ENV_VAR_NAME}`)
- Custom headers

### 2. Queries

Queries define what data to fetch and how to process it:

```toml
[queries.05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6]
id = "05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6"
aggregation_method = "median"
min_responses = 2
response_type = "ufixed256x18"
```

Each query includes:
- A unique identifier (required for Layer reporting)
- Aggregation method (currently supports "median" only)
- Minimum number of valid responses required
- Response type (for Ethereum ABI encoding)
- One or more endpoint configurations

## Adding Endpoints to Queries

For each query, you can specify multiple endpoints to get data from different sources:

```toml
[[queries.05cddb6b67074aa61fcbe1d2fd5924e028bb699b506267df28c88f7deac4edc6.endpoints]]
endpoint_type = "coingecko"          # Must match an endpoint name from [endpoints]
params = { coin_id = "savings-dai" } # Values to replace placeholders in url_template
response_path = ["savings-dai", "usd"] # Path to extract the value from JSON response
```

## Example Use Case

The example configuration fetches sDAI and Tellor token prices from multiple sources, ensuring data reliability through aggregation.

## Adding New Queries

To add a new query:

1. Add query ID to the `[queries]` section
2. Configure at least one endpoint under the query
3. Specify how to extract the data via `response_path`
