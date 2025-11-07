# Local Chain Version Comparison Guide

This guide explains how to use the block timing analysis scripts to compare performance between v5.1.2 and v6.0.0 on a local chain.

## Overview

The scripts are **version-agnostic** - they work with any running chain by simply connecting to RPC and metrics endpoints. No code changes are needed between versions. The only requirement is that the scripts are accessible (either in both branches or in a shared location).

## Workflow

### Step 1: Start Local Chain with v5.1.2

1. **Checkout the instrumented v5.1.2 branch:**
   ```bash
   git checkout timing-analysis-v5.1.2
   ```

2. **Build the binary:**
   ```bash
   make build
   # or
   go build ./cmd/layerd
   ```

3. **Start a local chain:**
   ```bash
   # Option 1: Single node (simplest)
   ./start_scripts/start_one_node.sh
   
   # Option 2: Local devnet (docker-based)
   make local-devnet
   
   # Option 3: Two validators
   # Terminal 1:
   ./start_scripts/start_two_chains.sh
   # Terminal 2:
   ./start_scripts/start_bill.sh
   ```

4. **Verify the chain is running:**
   ```bash
   curl http://localhost:26657/status
   curl http://localhost:26660/metrics | head
   ```

### Step 2: Collect Data for v5.1.2

1. **Start the collector with v5.1.2 config:**
   ```bash
   python scripts/block_timing_collector.py \
     --config scripts/config/block_timing_config_v5.1.2.yaml \
     --duration 3600  # Collect for 1 hour, or omit for indefinite
   ```

   The collector will:
   - Create output directory: `block_timing_data/v5.1.2/`
   - Tag all data with `"version": "v5.1.2"` in the JSON
   - Save files like: `v5.1.2_block_timing_20250106_120000.jsonl`

2. **Let it run for your desired duration** (e.g., 1-2 hours to get good sample size)

3. **Stop the collector** (Ctrl+C) when done

### Step 3: Upgrade Local Chain to v6.0.0

1. **Stop the current chain** (Ctrl+C in the terminal running layerd)

2. **Checkout the instrumented v6.0.0 branch:**
   ```bash
   git checkout timing-analysis-v6.0.0
   ```

3. **Build the new binary:**
   ```bash
   make build
   ```

4. **Restart the chain** (it will continue from the same state):
   ```bash
   # Use the same start script as before
   ./start_scripts/start_one_node.sh
   # or
   make local-devnet
   ```

   **Note:** The chain state persists in `~/.layer/`, so the upgrade will happen automatically if you have upgrade handlers configured, or you can manually trigger an upgrade proposal.

### Step 4: Collect Data for v6.0.0

1. **Start the collector with v6.0.0 config:**
   ```bash
   python scripts/block_timing_collector.py \
     --config scripts/config/block_timing_config_v6.0.0.yaml \
     --duration 3600
   ```

   The collector will:
   - Create output directory: `block_timing_data/v6.0.0/`
   - Tag all data with `"version": "v6.0.0"` in the JSON
   - Save files like: `v6.0.0_block_timing_20250106_140000.jsonl`

2. **Let it run for the same duration** as v5.1.2 for fair comparison

### Step 5: Compare the Data

1. **Analyze each dataset individually:**
   ```bash
   # Analyze v5.1.2 data
   python scripts/analyze_block_timing.py \
     --input block_timing_data/v5.1.2/v5.1.2_block_timing_*.jsonl \
     --output v5.1.2_summary.json
   
   # Analyze v6.0.0 data
   python scripts/analyze_block_timing.py \
     --input block_timing_data/v6.0.0/v6.0.0_block_timing_*.jsonl \
     --output v6.0.0_summary.json
   ```

2. **Compare the two datasets:**
   ```bash
   python scripts/analyze_block_timing.py --compare \
     --baseline block_timing_data/v5.1.2/v5.1.2_block_timing_*.jsonl \
     --test block_timing_data/v6.0.0/v6.0.0_block_timing_*.jsonl
   ```

   This will show:
   - Block time differences
   - Consensus phase timing changes
   - Module execution time changes
   - Which modules changed the most

## Important Notes

### Script Availability

The scripts work from **any branch** - they just need to be accessible. Options:

1. **Keep scripts in both branches** (recommended for easy access):
   - The scripts are in `scripts/` which is typically not version-specific
   - Just ensure both branches have the latest version of the scripts

2. **Use scripts from a single branch:**
   - You can checkout one branch, run the scripts, and they'll work with any running chain
   - The scripts connect to endpoints, they don't depend on the branch code

3. **Use a shared scripts directory:**
   - If you want to keep scripts separate, you can symlink or copy them

### Configuration Files

The config files (`block_timing_config_v5.1.2.yaml` and `block_timing_config_v6.0.0.yaml`) are identical except for:
- `version.tag`: Tags the data with the version
- `monitoring.output_dir`: Keeps data separate by version

You can also use the default config and just change the version tag:
```bash
# Edit scripts/config/block_timing_config.yaml and set:
# version:
#   tag: "v5.1.2"  # or "v6.0.0"
```

### Local Chain Considerations

- **Port conflicts**: If you're running multiple chains, make sure ports don't conflict
- **State persistence**: Local chain state is in `~/.layer/` - clear it if you want a fresh start
- **Metrics**: Ensure Prometheus metrics are enabled (usually default in local chains)
- **Log file**: The log path `~/.layer/layer.log` should match where your chain writes logs

### Upgrade Height

If you want to collect data before and after a specific upgrade height:

1. **Collect v5.1.2 data up to upgrade height:**
   ```bash
   # Monitor until you see the upgrade happen
   python scripts/block_timing_collector.py \
     --config scripts/config/block_timing_config_v5.1.2.yaml
   ```

2. **After upgrade, switch to v6.0.0 collector:**
   ```bash
   # Stop v5.1.2 collector (Ctrl+C)
   # Start v6.0.0 collector
   python scripts/block_timing_collector.py \
     --config scripts/config/block_timing_config_v6.0.0.yaml
   ```

   The data will be automatically tagged with versions, making it easy to split at the upgrade height during analysis.

## Example Output

After running the comparison, you'll see output like:

```
================================================================================
COMPARATIVE ANALYSIS
================================================================================

Block Time Comparison:
  Baseline Mean: 6.100s
  Test Mean:     6.300s
  Difference:    +0.200s (+3.3%)

Consensus Time Comparison:
  Baseline: 2700.0ms
  Test:     2750.0ms
  Difference: +50.0ms

Module Execution Time Changes:

  EndBlocker Modules (Top 10 changes):
    oracle              :   180.5ms →   195.2ms (+14.7ms, +8.1%)
    bridge              :    15.7ms →    16.2ms (+0.5ms, +3.2%)
    dispute             :     8.0ms →     8.1ms (+0.1ms, +1.3%)
```

This clearly shows which modules or phases account for performance differences between versions.

## Troubleshooting

### Scripts not found in branch
- The scripts are in `scripts/` directory
- If missing, copy them from another branch or checkout the branch that has them
- Scripts are version-agnostic, so any version will work

### Port already in use
- Check what's using the port: `lsof -i :26657`
- Use different ports or stop conflicting processes
- Update config file with different ports if needed

### Metrics not accessible
- Check metrics are enabled: `grep "prometheus" ~/.layer/*/config/config.toml`
- Default port is 26660
- Test: `curl http://localhost:26660/metrics`

### Data files not found
- Check the output directory exists: `ls -la block_timing_data/`
- Files are created when first block is collected
- Ensure collector ran long enough to collect at least one block

