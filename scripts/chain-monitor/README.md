# Chain Monitor

HTTP RPC block poller for Layer chain events. Standalone module under `scripts/chain-monitor/`.

See [PLAN.md](./PLAN.md) for the full design.

**Current: Phase 4** — hardened with structured logging, Prometheus metrics, dry-run, fixture tests, and systemd docs.

## Features

- Multi-URL CometBFT HTTP RPC with failover
- Height-by-height processing (backoff on errors; never panics)
- Durable cursor file — restart resumes at last height + 1
- YAML rules → Discord embeds
- **Weak aggregate** alerts when `aggregate_power < ratio × validator voting power`
- **Missing important reporters** on every aggregate (structured log for AI review) via `IMPORTANT_REPORTERS` + LCD `get_reports_by_aggregate`; optional Discord field on weak aggregates via `enrich: [missing_reporters]` (monikers from `reporters_map.json`)
- **Block interval**, **RPC unhealthy**, and **ingest lag** signal rules
- Valset timestamp recording + daily schedule report
- Named channels, rate limits, dedupe, query-id enrichment
- `-dry-run` / `dry_run: true`
- Structured logging (`-log-level`, `-log-format=text|json`)
- Full alert text logged on every rule match (`msg=alert` with `text=...`), independent of Discord rate limits / dedupe (`discord=sent|skipped`)
- Optional `defaults.log_rate_limit` safety valve (default unlimited) so hot rules cannot fill the disk
- Prometheus metrics at `GET /metrics`

## Quick start

```bash
cd scripts/chain-monitor
cp example-config.yml config.yml
# edit rpc.urls, channels.*.webhook_url; set dry_run: false for real alerts

./run.sh ./config.yml
go run ./cmd/chain-monitor -config=./config.yml -dry-run -log-level=debug
go test ./...
```

Production: see [deploy/README.md](./deploy/README.md) for a systemd unit.

## Rule kinds

| `match.kind` | Trigger |
|--------------|---------|
| *(empty)* / event | Chain event `match.event_type` (+ optional `when.attr_uint_lt_ratio`) |
| `block_interval` | Normalized time between blocks > `when.max_interval` |
| `rpc_unhealthy` | Consecutive RPC failures ≥ `when.fail_threshold` |
| `ingest_lag` | Tip − cursor > `when.max_lag` |
| `schedule` | Daily at `when.daily_at` (valset frequency report) |

### Weak aggregate

```yaml
when:
  attr_uint_lt_ratio:
    attr: aggregate_power
    ratio: 0.666
    against: validator_power   # from Tendermint RPC validators
```

If validator power has never been fetched or is stale, weak-aggregate alerts are **skipped** (no fake default).

### Missing important reporters

Set env `IMPORTANT_REPORTERS` to a comma-separated list of reporter bech32 addresses.

On **every** `aggregate_report` event the monitor queries Cosmos REST (`api.url` or `LAYER_API_URL`) for reports in that aggregate. If any configured reporters did not submit, it logs:

```text
important reporters missing from aggregate  query_id=... missing_reporters=[...] query_type=SpotPrice
```

`query_type` is included when it can be decoded from event `query_data`. Reporter addresses are replaced with monikers from `enrichment.reporters_map` when available.

Weak-aggregate Discord alerts can also set `enrich: [missing_reporters]` to show the same gap in the embed (`_missing_reporters`).

`api.url` must be LCD / gRPC-gateway (often `:1317`), **not** Tendermint RPC (`:26657`).

Example map (`example-reporters-map.json`):

```json
{
  "addressToMonikerMap": {
    "tellor1...": "palmito-reporter-1"
  }
}
```

### Synthetic embed attrs

`_height`, `_time`, `_node`, `_source`, `_asset_pair`, `_validator_power`, `_power_pct`, `_missing_reporters`

## Health & metrics

| Path | Meaning |
|------|---------|
| `GET /healthz` | Process up |
| `GET /readyz` | Cursor ready + recently successful |
| `GET /status` | JSON snapshot |
| `GET /metrics` | Prometheus counters/gauges |

Notable metrics: `chain_monitor_blocks_processed_total`, `chain_monitor_alerts_sent_total`, `chain_monitor_ingest_lag`, `chain_monitor_validator_power`.

## Module

Own `go.mod` — does not modify the Layer root module.

## Tests

```bash
go test ./...
```

Critical rule fixtures live under `testdata/fixtures/*.yml`.
