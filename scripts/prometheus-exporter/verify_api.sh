#!/bin/bash
# Verify all Prometheus Exporter API routes against a running host.
#
# Usage:
#   ./verify_api.sh http://54.160.217.166:8080
#   ./verify_api.sh http://54.160.217.166:8080 my-api-key
#   API_PASSWORD=my-api-key ./verify_api.sh http://127.0.0.1:8080
#
# Exit code: 0 if all checks pass, 1 otherwise.

set -euo pipefail

HOST="${1:-}"
API_KEY="${2:-${API_PASSWORD:-admin123}}"

if [[ -z "$HOST" ]]; then
  echo "Usage: $0 <host> [api_key]"
  echo "  host format: http://<ip>:<port>   e.g. http://54.160.217.166:8080"
  echo "  api_key: optional; defaults to \$API_PASSWORD or admin123"
  exit 1
fi

HOST="${HOST%/}"

if ! command -v curl >/dev/null 2>&1; then
  echo "ERROR: curl is required"
  exit 1
fi

PASS=0
FAIL=0

# check NAME EXPECT_HTTP KIND METHOD PATH [curl args...]
# KIND: json_ok | csv_ok | json_err
check() {
  local name="$1"
  local expect_http="$2"
  local kind="$3"
  local method="$4"
  local path="$5"
  shift 5

  local url="${HOST}${path}"
  local tmp errf
  tmp="$(mktemp)"
  errf="$(mktemp)"
  local http_code curl_rc=0
  http_code="$(
    curl -sS -o "$tmp" -w "%{http_code}" \
      -X "$method" \
      --connect-timeout 10 \
      --max-time 60 \
      "$@" \
      "$url" 2>"$errf"
  )" || curl_rc=$?

  # curl writes "000" on connect failure; don't append another code on top.
  if [[ -z "$http_code" || "$http_code" == "000" ]]; then
    http_code="000"
  fi

  local body
  body="$(cat "$tmp")"
  local err
  err="$(tr '\n' ' ' <"$errf" | sed 's/[[:space:]]*$//')"
  rm -f "$tmp" "$errf"

  local ok=0
  if [[ "$http_code" == "$expect_http" ]]; then
    case "$kind" in
      json_ok)
        [[ "$body" == *'"success":true'* ]] && ok=1
        ;;
      json_err)
        [[ "$body" == *'"success":false'* ]] && ok=1
        ;;
      csv_ok)
        # CSV: non-empty body and not a JSON error payload
        if [[ -n "$body" && "$body" != *'"success":false'* ]]; then
          ok=1
        fi
        ;;
    esac
  fi

  if [[ "$ok" -eq 1 ]]; then
    PASS=$((PASS + 1))
    echo "PASS  [$http_code] $name"
  else
    FAIL=$((FAIL + 1))
    if [[ "$http_code" == "000" ]]; then
      echo "FAIL  [connection failed] $name"
      echo "      could not reach ${url}"
      [[ -n "$err" ]] && echo "      curl: $err (exit $curl_rc)"
    else
      echo "FAIL  [$http_code expected $expect_http] $name"
      echo "      body: ${body:0:300}"
    fi
  fi
}

echo "Verifying Prometheus Exporter API"
echo "  Host: $HOST"
echo "  Key:  ${API_KEY:0:2}***"
echo ""

AUTH=(-H "X-API-Key: $API_KEY")
JSON=(-H "Content-Type: application/json")

# --- Positive route checks ---
check "GET  /api/health"                        200 json_ok GET  "/api/health"

check "GET  /api/prices?limit=5"                200 json_ok GET  "/api/prices?limit=5" \
  "${AUTH[@]}"

check "GET  /api/prices/latest"                 200 json_ok GET  "/api/prices/latest" \
  "${AUTH[@]}"

check "GET  /api/prices/market/BTC-USD"         200 json_ok GET  "/api/prices/market/BTC-USD?limit=5" \
  "${AUTH[@]}"

check "GET  /api/prices/exchange/Bitfinex"      200 json_ok GET  "/api/prices/exchange/Bitfinex?limit=5" \
  "${AUTH[@]}"

check "GET  /api/prices/range"                  200 json_ok GET  "/api/prices/range?start=2020-01-01&end=2030-01-01&limit=5" \
  "${AUTH[@]}"

check "GET  /api/prices/latest?format=csv"      200 csv_ok GET  "/api/prices/latest?format=csv" \
  "${AUTH[@]}"

check "POST /api/query (SELECT)"                200 json_ok POST "/api/query" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{"sql":"SELECT market_id, COUNT(*) AS n FROM price_data GROUP BY 1 ORDER BY 1 LIMIT 20"}'

check "POST /api/query (CTE)"                   200 json_ok POST "/api/query" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{"sql":"WITH t AS (SELECT * FROM price_data LIMIT 100) SELECT exchange_id, AVG(price) AS avg_price FROM t GROUP BY 1"}'

check "POST /api/query (CSV)"                   200 csv_ok POST "/api/query" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{"sql":"SELECT market_id, exchange_id, price FROM price_data LIMIT 5","format":"csv"}'

# --- Negative / security checks ---
check "GET  /api/prices (bad API key → 401)"    401 json_err GET  "/api/prices?limit=1" \
  -H "X-API-Key: wrongpassword"

check "POST /api/query (no auth → 401)"         401 json_err POST "/api/query" \
  "${JSON[@]}" \
  -d '{"sql":"SELECT 1"}'

check "POST /api/query (DELETE rejected → 400)" 400 json_err POST "/api/query" \
  "${AUTH[@]}" "${JSON[@]}" \
  -d '{"sql":"DELETE FROM price_data"}'

echo ""
echo "----------------------------------------"
echo "Results: $PASS passed, $FAIL failed (of $((PASS + FAIL)))"
echo "----------------------------------------"

if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
exit 0
