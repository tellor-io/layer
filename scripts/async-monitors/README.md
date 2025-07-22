# Async Monitor Events

This script provides an alternative to the WebSocket-based monitor that uses HTTP RPC queries instead of WebSocket connections to monitor blockchain events.

## Features

- **HTTP RPC-based monitoring**: Uses HTTP requests instead of WebSocket connections
- **Block-by-block processing**: Queries for each block sequentially
- **Rate limiting**: Ensures only 4 requests per second
- **Event monitoring**: Monitors all configured event types from the config file
- **Block time monitoring**: Alerts when block times exceed thresholds
- **Validator power tracking**: Updates total reporter power every 10 minutes
- **Timestamp analysis**: Optional daily analysis of validator set updates
- **Health checks**: Monitors node responsiveness
- **Graceful shutdown**: Handles OS signals for clean shutdown

## Usage

```bash
go run ./scripts/async-monitors/async-monitor-events.go \
  -rpc-url=<rpc_url> \
  -config=<config_file_path> \
  -node=<node_name> \
  [-block-time-threshold=<duration>] \
  [-timestamp-analyzer]
```

### Parameters

- `-rpc-url`: RPC URL (default: 127.0.0.1:26657)
- `-config`: Path to the event configuration file (required)
- `-node`: Name of the node being monitored (required)
- `-block-time-threshold`: Block time threshold (e.g., 5m, 1h). If not set, block time monitoring is disabled
- `-timestamp-analyzer`: Enable analyzer of validator set update timestamps

### Example

```bash
go run ./scripts/async-monitors/async-monitor-events.go \
  -rpc-url=127.0.0.1:26657 \
  -config=../monitors/event-config.yml \
  -node=my-node \
  -block-time-threshold=5m \
  -timestamp-analyzer
```

## How It Works

1. **Initialization**: 
   - Loads the event configuration from the specified YAML file
   - Gets the latest block height from the RPC endpoint
   - Initializes monitoring from that block height

2. **Block Monitoring Loop**:
   - Continuously queries for the next block (current height + 1)
   - If the block is available, processes it and increments the block height
   - If the block is not available, waits 1 second and tries again
   - Maintains a minimum 1-second interval between requests

3. **Event Processing**:
   - Extracts events from `begin_block`, `end_block`, and transaction results
   - Matches events against configured event types
   - Sends Discord alerts for matching events
   - Handles special cases like aggregate reports with rate limiting

4. **Health Monitoring**:
   - Performs health checks every 30 seconds
   - Sends liveness alerts if the node becomes unresponsive
   - Updates validator power every 10 minutes

## Configuration

The script uses the same configuration file as the WebSocket monitor (`event-config.yml`). The configuration defines:

- Event types to monitor
- Alert names and types
- Webhook URLs for Discord notifications

## Differences from WebSocket Monitor

| Feature | WebSocket Monitor | Async Monitor |
|---------|------------------|---------------|
| Connection | WebSocket subscription | HTTP RPC queries |
| Block processing | Real-time events | Sequential block queries |
| Rate limiting | None | 1-second minimum interval |
| Reconnection | Automatic WebSocket reconnection | HTTP retry logic |
| Resource usage | Lower (persistent connection) | Higher (frequent HTTP requests) |
| Reliability | Depends on WebSocket stability | More reliable for unstable networks |

## Error Handling

- **Network errors**: Retries with exponential backoff
- **Invalid responses**: Logs errors and continues monitoring
- **Configuration errors**: Logs errors and uses default values where possible
- **Panic recovery**: Catches panics and sends crash alerts

## Output Files

- `bridge_validator_timestamps.csv`: Tracks validator set update timestamps for analysis

## Dependencies

- `github.com/tellor-io/layer/utils`: For Discord notifications
- `gopkg.in/yaml.v3`: For configuration file parsing
- Standard Go libraries: `net/http`, `encoding/json`, `time`, etc. 