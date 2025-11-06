# Per-Block Timing Analysis System

This system provides comprehensive per-block timing analysis for the Tellor Layer blockchain, tracking consensus phases, module execution times, and transaction details to identify performance bottlenecks and correlate block time spikes with specific events (like tips).

## Overview

The system consists of:
1. **Data Collection Script** (`block_timing_collector.py`) - Monitors the chain and collects timing data
2. **Analysis Script** (`analyze_block_timing.py`) - Analyzes collected data and generates insights
3. **Configuration File** (`config/block_timing_config.yaml`) - Configurable endpoints and settings
4. **Instrumented Branches** - Two branches with minimal timing instrumentation for comparison

## Quick Start

### 1. Prerequisites

```bash
# Python 3.7+ required
pip install pyyaml requests

# Ensure your node is running with metrics enabled
# Check http://localhost:26660/metrics is accessible
```

### 2. Start Collecting Data

```bash
# Using default config
python scripts/block_timing_collector.py

# With custom config
python scripts/block_timing_collector.py --config /path/to/config.yaml

# For a specific duration (e.g., 1 hour)
python scripts/block_timing_collector.py --duration 3600
```

The collector will:
- Poll consensus state every 200ms
- Scrape metrics on each new block
- Calculate per-block deltas for module timing
- Parse transactions to identify tips
- Output data to `block_timing_data/block_timing_YYYYMMDD_HHMMSS.jsonl`

### 3. Analyze Collected Data

```bash
# Analyze a data file
python scripts/analyze_block_timing.py --input block_timing_data/block_timing_20251106_103000.jsonl

# Export summary statistics
python scripts/analyze_block_timing.py --input block_timing_data/block_timing_20251106_103000.jsonl \
  --output summary.json

# Compare two datasets (e.g., v5.1.2 vs v6.0.0)
python scripts/analyze_block_timing.py --compare \
  --baseline block_timing_data/v5.1.2_data.jsonl \
  --test block_timing_data/v6.0.0_data.jsonl
```

## Data Collection Details

### Data Sources

The collector combines multiple data sources for comprehensive analysis:

1. **Consensus State** (`/consensus_state` endpoint)
   - Tracks consensus phase transitions (Propose → Prevote → Precommit → Commit)
   - Records timing for each phase
   - Detects multiple rounds (timeouts)

2. **Prometheus Metrics** (`/metrics` endpoint)
   - `begin_blocker_sum{module="X"}` - Cumulative BeginBlocker time per module
   - `end_blocker_sum{module="X"}` - Cumulative EndBlocker time per module
   - Delta calculation provides per-block timing

3. **Block Data** (RPC endpoints)
   - `/block?height=N` - Block header, transactions, proposer
   - `/block_results?height=N` - Execution results, gas used, events
   - Transaction parsing identifies message types and tips

4. **Application Logs** (optional, with instrumented branches)
   - `[ABCI_TIMING]` entries from instrumented code
   - Total FinalizeBlock duration

### Output Format

Each line in the output JSONL file represents one block:

```json
{
  "height": 12345,
  "timestamp": "2025-11-06T10:30:45.123456Z",
  "proposer": "cosmosvalcons1abc...",
  "total_block_time_seconds": 6.342,
  
  "consensus": {
    "rounds": 0,
    "propose_duration_ms": 1200,
    "prevote_duration_ms": 800,
    "precommit_duration_ms": 900,
    "commit_duration_ms": 50,
    "total_consensus_ms": 2950,
    "percent_of_block_time": 46.5
  },
  
  "execution": {
    "begin_block_modules": {
      "mint": 12.3,
      "oracle": 5.1,
      "reporter": 2.0,
      "total": 19.4
    },
    "end_block_modules": {
      "oracle": 185.4,
      "dispute": 8.2,
      "bridge": 15.7,
      "reporter": 10.5,
      "total": 219.8
    },
    "total_execution_ms": 239.2,
    "percent_of_block_time": 3.8
  },
  
  "transactions": {
    "count": 5,
    "gas_used": 250000,
    "gas_wanted": 300000,
    "message_types": {
      "MsgTip": 2,
      "MsgSubmitValue": 1,
      "MsgDelegate": 2
    },
    "tips": [
      {"query_id": "0x01", "amount": "1000loya", "tipper": "cosmos1..."}
    ]
  },
  
  "abci": {
    "finalize_block_ms": 280
  },
  
  "analysis": {
    "has_tips": true,
    "tip_count": 2
  }
}
```

## Analysis Features

The analysis script provides:

### 1. Summary Statistics
- Block time distribution (mean, median, min, max, std dev)
- Consensus phase timing breakdown
- Per-module execution times
- Tip correlation analysis

### 2. Slow Block Identification
- Flags blocks >2 std dev above mean (configurable)
- Shows which modules were slow in those blocks
- Correlates with tip events

### 3. Comparative Analysis
- Compare two datasets (e.g., different SDK versions)
- Shows differences in block time, consensus time, module times
- Identifies which modules changed the most

### Example Output

```
================================================================================
BLOCK TIMING ANALYSIS SUMMARY
================================================================================

Analysis Period:
  Heights: 12000 - 12500
  Total Blocks: 500

Block Time Statistics:
  Mean:     6.100s
  Median:   6.000s
  Min:      5.800s
  Max:      8.200s
  Std Dev:  0.400s

Consensus Statistics:
  Mean Propose:    1100.0ms
  Mean Prevote:    750.0ms
  Mean Precommit:  850.0ms
  Total Consensus: 2700.0ms
  Blocks with Multiple Rounds: 5

EndBlocker Module Times (mean):
  oracle              :   180.5ms (max: 450.2ms)
  bridge              :    15.7ms (max: 45.3ms)
  dispute             :     8.0ms (max: 25.1ms)
  reporter            :    10.2ms (max: 18.5ms)

Tip Correlation Analysis:
  Blocks with Tips:    45
  Blocks without Tips: 455
  Avg Time (with tips):    6.800s
  Avg Time (without tips): 6.000s
  Difference:              0.800s (13.3% increase)

================================================================================
SLOW BLOCKS (3 blocks identified)
================================================================================

1. Block 12345:
   Time: 8.200s (5.25 std devs above mean)
   Tips: YES (count: 15)
   Txs: 25
   Slowest Module: end_blocker.oracle (450.2ms)

2. Block 12378:
   Time: 7.850s (4.38 std devs above mean)
   Tips: YES (count: 8)
   Txs: 18
   Slowest Module: end_blocker.oracle (380.5ms)
```

## Instrumented Branches

Two branches have been created with minimal timing instrumentation:

### Branch 1: `timing-analysis-v5.1.2` (Cosmos SDK v0.50.9)
### Branch 2: `timing-analysis-v6.0.0` (Cosmos SDK v0.53.4)

Both branches include a single file: `app/abci_timing.go` with a FinalizeBlock wrapper that logs timing information.

### Using Instrumented Branches

```bash
# Build and run v5.1.2 instrumented version
git checkout timing-analysis-v5.1.2
make build
# Deploy to test environment

# Collect data
python scripts/block_timing_collector.py --duration 3600 \
  --config config_v5.1.2.yaml

# Build and run v6.0.0 instrumented version
git checkout timing-analysis-v6.0.0
make build
# Deploy to test environment

# Collect data
python scripts/block_timing_collector.py --duration 3600 \
  --config config_v6.0.0.yaml

# Compare the two datasets
python scripts/analyze_block_timing.py --compare \
  --baseline block_timing_data/v5.1.2_data.jsonl \
  --test block_timing_data/v6.0.0_data.jsonl
```

### Instrumentation Details

The instrumentation adds **ONLY** timing/logging code with **NO** business logic changes:

```go
func (app *App) FinalizeBlock(req *abci.FinalizeBlockRequest) (*abci.FinalizeBlockResponse, error) {
    startTime := time.Now()
    resp, err := app.BaseApp.FinalizeBlock(req)  // All logic happens here
    
    app.Logger().Info("[ABCI_TIMING]",
        "height", req.Height,
        "finalize_block_ms", time.Since(startTime).Milliseconds(),
        "num_txs", len(req.Txs),
    )
    
    return resp, err
}
```

The app hash remains **identical** to the non-instrumented version.

## Configuration

Edit `scripts/config/block_timing_config.yaml`:

```yaml
endpoints:
  rpc: "http://localhost:26657"
  metrics: "http://localhost:26660/metrics"
  websocket: "ws://localhost:26657/websocket"

monitoring:
  poll_interval_ms: 200              # Consensus state polling frequency
  output_dir: "./block_timing_data"  # Where to save data
  enable_websocket: true             # Use websocket for block events
  log_file_path: "~/.layer/layer.log"  # Application log file

analysis:
  slow_block_threshold_std_dev: 2.0  # Threshold for flagging slow blocks
  moving_average_window: 20          # Window for moving averages

filters:
  min_height: null                   # Start from current if null
  max_height: null                   # Run indefinitely if null
  track_message_types:               # Message types to track
    - "MsgTip"
    - "MsgSubmitValue"
    - "MsgDelegate"
```

## Use Cases

### 1. Investigating Block Time Increase After Upgrade

**Problem**: After upgrading from SDK v0.50.9 to v0.53.4, block time increased by 0.2s.

**Solution**:
```bash
# Collect data from v5.1.2 node
python scripts/block_timing_collector.py --config config_v5.1.2.yaml --duration 7200

# Collect data from v6.0.0 node
python scripts/block_timing_collector.py --config config_v6.0.0.yaml --duration 7200

# Compare
python scripts/analyze_block_timing.py --compare \
  --baseline block_timing_data/v5.1.2_*.jsonl \
  --test block_timing_data/v6.0.0_*.jsonl
```

**Result**: Identifies which module(s) or consensus phase accounts for the 0.2s increase.

### 2. Correlating Tips with Block Time Spikes

**Problem**: Do multiple tips in a block cause block time spikes?

**Solution**:
```bash
# Collect data during period with tip activity
python scripts/block_timing_collector.py --duration 3600

# Analyze
python scripts/analyze_block_timing.py --input block_timing_data/block_timing_*.jsonl
```

**Result**: See "Tip Correlation Analysis" showing average block time difference and identify which module slows down during tip blocks.

### 3. Identifying Performance Bottlenecks

**Problem**: Which module takes the most time in block execution?

**Solution**:
```bash
python scripts/analyze_block_timing.py --input block_timing_data/block_timing_*.jsonl
```

**Result**: See per-module timing breakdown showing mean, median, max times.

## Troubleshooting

### Metrics endpoint not accessible
```bash
# Check if metrics are enabled in config.toml
grep "prometheus = true" ~/.layer/config/config.toml

# Check if port is open
curl http://localhost:26660/metrics
```

### Consensus timing missing
- Consensus state tracking requires frequent polling
- Ensure `poll_interval_ms` is set to 200ms or less
- Consensus timing may be incomplete for very fast blocks

### Module timing deltas are negative or zero
- This occurs if metrics are scraped at the exact same block height
- The collector handles this by ignoring negative deltas
- Ensure you're monitoring for multiple blocks

### ABCI timing not appearing
- Only available when using instrumented branches
- Ensure `log_file_path` points to correct log file
- Check that log level is INFO or lower
- Verify `[ABCI_TIMING]` entries appear in logs: `grep "ABCI_TIMING" ~/.layer/layer.log`

## Performance Impact

- **Collector script**: Negligible (runs externally, only polls RPC/metrics)
- **Instrumented code**: <1ms per block (single log write)
- **Consensus state polling**: ~5 requests/second at 200ms intervals

## Notes

- The collector is designed to run continuously without missing blocks
- Output files are in JSONL format (one JSON object per line) for easy streaming processing
- All times are in milliseconds unless otherwise specified
- The system works with both v0.50.9 and v0.53.4 Cosmos SDK versions

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Verify your node's RPC and metrics endpoints are accessible
3. Review the example output formats in this README

