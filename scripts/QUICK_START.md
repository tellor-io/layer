# Block Timing Analysis - Quick Start

## Prerequisites

```bash
pip install pyyaml requests
```

## Basic Usage

### 1. Start Collecting Data

```bash
cd /Users/caleb/layer-repo

# Collect for 1 hour
python scripts/block_timing_collector.py --duration 3600

# Collect indefinitely (Ctrl+C to stop)
python scripts/block_timing_collector.py
```

Output: `block_timing_data/block_timing_YYYYMMDD_HHMMSS.jsonl`

### 2. Analyze Data

```bash
# View summary
python scripts/analyze_block_timing.py --input block_timing_data/block_timing_*.jsonl

# Export to JSON
python scripts/analyze_block_timing.py \
  --input block_timing_data/block_timing_*.jsonl \
  --output summary.json
```

### 3. Compare Two Datasets

```bash
python scripts/analyze_block_timing.py --compare \
  --baseline block_timing_data/v5_data.jsonl \
  --test block_timing_data/v6_data.jsonl
```

## Using Instrumented Branches

### v5.1.2 Branch (SDK v0.50.9)

```bash
git checkout timing-analysis-v5.1.2
make build
# Deploy layerd binary to test node
# Start node and collect data
python scripts/block_timing_collector.py --duration 3600
```

### v6.0.0 Branch (SDK v0.53.4)

```bash
git checkout timing-analysis-v6.0.0
make build
# Deploy layerd binary to test node  
# Start node and collect data
python scripts/block_timing_collector.py --duration 3600
```

## Configuration

Edit `scripts/config/block_timing_config.yaml`:

```yaml
endpoints:
  rpc: "http://localhost:26657"
  metrics: "http://localhost:26660/metrics"

monitoring:
  poll_interval_ms: 200
  output_dir: "./block_timing_data"
  log_file_path: "~/.layer/layer.log"
```

## Troubleshooting

### Check Metrics Availability
```bash
curl http://localhost:26660/metrics | grep begin_blocker
```

### Check RPC Endpoint
```bash
curl http://localhost:26657/status
```

### Enable Metrics (if not working)
```bash
# In ~/.layer/config/config.toml:
[instrumentation]
prometheus = true
prometheus_listen_addr = ":26660"
```

## Output Examples

### Per-Block Data (JSONL)
Each line is one block with complete timing breakdown:
- Block time, height, timestamp
- Consensus phase timing (propose, prevote, precommit)
- Module execution times (begin/end blocker per module)
- Transaction details and tips

### Analysis Summary
- Block time statistics (mean, median, std dev)
- Consensus timing breakdown
- Per-module execution times
- Tip correlation (avg time with/without tips)
- Slow block identification

## Key Branches

- **`timing-analysis-v5.1.2`** - Instrumented v5.1.2 (SDK v0.50.9)
- **`timing-analysis-v6.0.0`** - Instrumented v6.0.0 (SDK v0.53.4)
- **`feat/persistent-heartbeat-data`** - Your current branch (unchanged)

Both instrumented branches add only `app/abci_timing.go` with a FinalizeBlock wrapper for timing logs.

## What You Get

✅ Per-block timing breakdown  
✅ Consensus vs execution split  
✅ Module-level execution times  
✅ Tip correlation analysis  
✅ Slow block identification  
✅ SDK version comparison  

## Full Documentation

- **`IMPLEMENTATION_SUMMARY.md`** - What was implemented and why
- **`BLOCK_TIMING_README.md`** - Comprehensive user guide
- **`QUICK_START.md`** - This file

## Example Workflow

```bash
# 1. Collect data from current node
python scripts/block_timing_collector.py --duration 7200

# 2. Analyze
python scripts/analyze_block_timing.py \
  --input block_timing_data/block_timing_20251106_*.jsonl

# 3. Review slow blocks and tip correlation
# Output shows which modules are slow and if tips cause spikes

# 4. Optional: Compare with v5.1.2 by building/deploying that branch
#    and collecting data there too
```

## Support

Questions? Check:
1. **BLOCK_TIMING_README.md** - Detailed documentation
2. **IMPLEMENTATION_SUMMARY.md** - Implementation details
3. Output format examples in README

