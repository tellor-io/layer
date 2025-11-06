# Block Timing Analysis Implementation Summary

## What Was Implemented

I've successfully implemented a comprehensive per-block timing analysis system for your Cosmos SDK chain. This system will help you investigate the 0.2s block time increase after upgrading from SDK v0.50.9 to v0.53.4 and identify if tips cause block time spikes.

## Files Created

### 1. Data Collection & Analysis Scripts
- **`scripts/block_timing_collector.py`** (585 lines)
  - Monitors consensus state transitions in real-time
  - Scrapes Prometheus metrics and calculates per-block deltas
  - Analyzes transactions to identify tips
  - Parses application logs for ABCI timing
  - Outputs detailed per-block data in JSONL format

- **`scripts/analyze_block_timing.py`** (442 lines)
  - Statistical analysis of collected data
  - Identifies slow blocks (>2 std dev above mean)
  - Tip correlation analysis
  - Comparative analysis between two datasets
  - Exports summary statistics

### 2. Configuration
- **`scripts/config/block_timing_config.yaml`**
  - RPC/metrics endpoint configuration
  - Polling intervals and output directories
  - Analysis thresholds
  - Message type tracking

### 3. Documentation
- **`scripts/BLOCK_TIMING_README.md`** (comprehensive user guide)
- **`scripts/IMPLEMENTATION_SUMMARY.md`** (this file)

### 4. Instrumented Branches
- **`timing-analysis-v5.1.2`** - Branched from tags/v5.1.2 (SDK v0.50.9)
- **`timing-analysis-v6.0.0`** - Branched from tags/v6.0.0 (SDK v0.53.4)

Both branches include `app/abci_timing.go` with a FinalizeBlock wrapper that logs timing information **without modifying any business logic**.

## How It Works

### Data Collection Flow

1. **Consensus State Monitoring** (200ms polling)
   - Tracks phase transitions: Propose → Prevote → Precommit → Commit
   - Records duration of each phase
   - Detects timeouts/multiple rounds

2. **Metrics Scraping** (per new block)
   - Fetches `begin_blocker_sum{module="X"}` and `end_blocker_sum{module="X"}`
   - Calculates delta from previous block to get per-block timing
   - Provides module-level execution breakdown

3. **Transaction Analysis**
   - Parses block transactions and results
   - Identifies message types (MsgTip, MsgSubmitValue, etc.)
   - Extracts tip details (query_id, amount, tipper)

4. **Log Parsing** (optional, with instrumented branches)
   - Reads `[ABCI_TIMING]` entries from application logs
   - Provides total FinalizeBlock duration

### Per-Block Output

Each block produces a comprehensive JSON record with:
- Total block time
- Consensus breakdown (propose, prevote, precommit timing)
- Module execution times (BeginBlocker and EndBlocker per module)
- Transaction details and tip information
- Analysis flags (is_slow, has_tips)

## Quick Start Guide

### Step 1: Test the Collector (No Code Changes)

You can start collecting data immediately on your current v6.0.0 node:

```bash
# Make sure your node's metrics are accessible
curl http://localhost:26660/metrics | grep begin_blocker

# Start collecting data for 1 hour
cd /Users/caleb/layer-repo
python scripts/block_timing_collector.py --duration 3600
```

This will create a file like `block_timing_data/block_timing_20251106_HHMMSS.jsonl`.

### Step 2: Analyze the Data

```bash
# View summary statistics
python scripts/analyze_block_timing.py --input block_timing_data/block_timing_*.jsonl

# Export summary to JSON
python scripts/analyze_block_timing.py \
  --input block_timing_data/block_timing_*.jsonl \
  --output summary.json
```

### Step 3: (Optional) Use Instrumented Branches for Enhanced Data

If you want even more detailed timing data:

```bash
# Build v6.0.0 instrumented version
git checkout timing-analysis-v6.0.0
make build

# Deploy to your test node and collect data
python scripts/block_timing_collector.py --duration 3600

# For comparison, also collect from v5.1.2
git checkout timing-analysis-v5.1.2
make build
# Deploy to test node and collect data
```

Then compare:

```bash
python scripts/analyze_block_timing.py --compare \
  --baseline block_timing_data/v5.1.2_data.jsonl \
  --test block_timing_data/v6.0.0_data.jsonl
```

## What You'll Learn

### 1. Where the 0.2s Increase Comes From

The comparative analysis will show:
- Is it in consensus time? (propose/prevote/precommit phases)
- Is it in execution time? (begin/end blockers)
- Which specific module(s) got slower?

Example output:
```
Block Time Comparison:
  Baseline Mean: 6.000s
  Test Mean:     6.200s
  Difference:    +0.200s (+3.3%)

Module Execution Time Changes:
  EndBlocker Modules (Top changes):
    oracle              : 180.0ms → 210.0ms (+30.0ms, +16.7%)
    bridge              :  15.0ms →  20.0ms (+5.0ms, +33.3%)
```

### 2. Do Tips Cause Block Time Spikes?

The tip correlation analysis will show:
```
Tip Correlation Analysis:
  Blocks with Tips:    45
  Blocks without Tips: 455
  Avg Time (with tips):    6.800s
  Avg Time (without tips): 6.000s
  Difference:              0.800s (13.3% increase)
```

Plus, you'll see which specific module slows down during tip blocks:
```
SLOW BLOCKS:
1. Block 12345:
   Time: 8.200s
   Tips: YES (count: 15)
   Slowest Module: end_blocker.oracle (450.2ms)
```

### 3. Performance Bottlenecks

Per-module breakdown shows where time is spent:
```
EndBlocker Module Times (mean):
  oracle    : 180.5ms (max: 450.2ms)  <-- Highest variance!
  bridge    :  15.7ms (max: 45.3ms)
  dispute   :   8.0ms (max: 25.1ms)
  reporter  :  10.2ms (max: 18.5ms)
```

## Key Features

### 1. Zero-Risk Data Collection
- The collector runs **externally** and only reads data
- No code changes needed for basic analysis
- Works on production nodes

### 2. Per-Block Granularity
- Every single block gets its own record
- Can correlate exact blocks with tip events
- No averaging - see actual spikes

### 3. Comprehensive Timing Breakdown
- Consensus phases (propose, prevote, precommit)
- Module execution (begin/end blocker per module)
- Transaction details and gas usage
- Optional ABCI timing from instrumented code

### 4. Comparative Analysis
- Compare two different SDK versions
- Compare different time periods
- Identify performance regressions

## Safety Notes on Instrumented Branches

The instrumented branches add **only** timing/logging code:

```go
// This is the ONLY change
func (app *App) FinalizeBlock(req *abci.FinalizeBlockRequest) (*abci.FinalizeBlockResponse, error) {
    startTime := time.Now()
    resp, err := app.BaseApp.FinalizeBlock(req)  // Original logic untouched
    app.Logger().Info("[ABCI_TIMING]", "height", req.Height, "finalize_block_ms", time.Since(startTime).Milliseconds(), "num_txs", len(req.Txs))
    return resp, err
}
```

- ✅ No business logic modified
- ✅ App hash remains identical
- ✅ Single log write per block (<1ms overhead)
- ✅ Can safely run on testnet for testing

## What to Do Next

### Immediate Action: Collect Data from Current Setup

Run the collector on your current v6.0.0 node during normal operation and during tip activity:

```bash
cd /Users/caleb/layer-repo
python scripts/block_timing_collector.py --duration 7200  # 2 hours
```

This will give you baseline data showing:
- Current block time distribution
- Which modules are slow
- Impact of tips on block time

### Follow-up: Compare SDK Versions

If you want to definitively identify what changed between v0.50.9 and v0.53.4:

1. Build and deploy timing-analysis-v5.1.2 to a test node
2. Collect 1-2 hours of data
3. Build and deploy timing-analysis-v6.0.0 to a test node
4. Collect 1-2 hours of data
5. Run comparative analysis

The comparison will show exactly which module(s) account for the 0.2s increase.

## Configuration Options

Edit `scripts/config/block_timing_config.yaml` to adjust:
- RPC/metrics endpoints (if not on localhost)
- Polling interval (default 200ms)
- Output directory
- Log file path (for instrumented branches)
- Slow block threshold
- Message types to track

## Troubleshooting

### Metrics endpoint not working?
```bash
# Enable in config.toml
grep "prometheus = true" ~/.layer/config/config.toml

# Restart node if needed
```

### Want to verify metrics before running collector?
```bash
# Check if metrics are available
curl http://localhost:26660/metrics | grep -E "begin_blocker|end_blocker"

# Should see lines like:
# begin_blocker_sum{module="mint"} 12345.0
# end_blocker_sum{module="oracle"} 67890.0
```

### Collector not detecting new blocks?
```bash
# Check RPC endpoint
curl http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Make sure it's incrementing
```

## Expected Output Files

After running for a while, you'll have:

```
block_timing_data/
  block_timing_20251106_103000.jsonl  # Per-block data (one JSON per line)
  block_timing_20251106_110000.jsonl  # Each run creates new file
```

And after analysis:

```
summary.json                          # Exported summary statistics
```

Each JSONL file contains one line per block with complete timing breakdown.

## Questions This System Answers

1. ✅ **Where did the 0.2s increase come from?**
   - Run collector on both SDK versions and compare

2. ✅ **Do tips cause block time spikes?**
   - Tip correlation analysis shows average impact

3. ✅ **Which blocks are slowest?**
   - Slow block identification with details

4. ✅ **Which module is the bottleneck?**
   - Per-module timing breakdown

5. ✅ **Is it consensus or execution that's slow?**
   - Consensus vs execution breakdown with percentages

## Support

- Full documentation: `scripts/BLOCK_TIMING_README.md`
- Example usage patterns in README
- Troubleshooting section for common issues

The system is ready to use! Start with the basic collector (no code changes) to get immediate insights, then optionally use the instrumented branches for even more detailed analysis.

