# Deploying with systemd

Example unit: [`deploy/chain-monitor.service`](./deploy/chain-monitor.service).

## Layout

| Path | Purpose |
|------|---------|
| `/opt/chain-monitor/chain-monitor` | Built binary |
| `/etc/chain-monitor/config.yml` | Config (webhooks, RPC URLs) |
| `/var/lib/chain-monitor/` | Cursor + valset timestamp files (`state.*` paths) |

## Install sketch

```bash
# build
cd scripts/chain-monitor
go build -o chain-monitor ./cmd/chain-monitor

sudo useradd --system --home /var/lib/chain-monitor --shell /usr/sbin/nologin chain-monitor
sudo mkdir -p /opt/chain-monitor /etc/chain-monitor /var/lib/chain-monitor
sudo cp chain-monitor /opt/chain-monitor/
sudo cp config.yml /etc/chain-monitor/config.yml
sudo chown -R chain-monitor:chain-monitor /var/lib/chain-monitor
sudo cp deploy/chain-monitor.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now chain-monitor
```

Point `state.cursor_path` and `state.valset_timestamps_path` under `/var/lib/chain-monitor/`.

Health / metrics (default `health.listen: ":8080"`):

- `GET /healthz` — liveness
- `GET /readyz` — readiness
- `GET /status` — JSON snapshot
- `GET /metrics` — Prometheus text

Dry-run first: set `dry_run: true` (or pass `-dry-run`) and confirm embeds in journald before enabling Discord webhooks.

## Missing reporters (optional)

When `IMPORTANT_REPORTERS` is set, every `aggregate_report` is checked via LCD and gaps are logged (`important reporters missing from aggregate`). Weak-aggregate Discord alerts can also use `enrich: [missing_reporters]`.

1. Copy `example-reporters-map.json` → your deploy path and set `enrichment.reporters_map`.
2. Set `api.url` in config (or `LAYER_API_URL`) to the Cosmos REST/LCD base — **not** `:26657`.
3. Export `IMPORTANT_REPORTERS=tellor1...,tellor1...` in the systemd unit (`Environment=` or `EnvironmentFile=`).
