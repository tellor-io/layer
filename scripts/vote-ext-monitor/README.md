# Vote Extension Participation Rate Monitor

This monitor tracks vote extension participation rates and verifies signatures in vote extensions for the Layer blockchain.

## Features

### 1. Participation Rate Monitoring
- Tracks vote extension participation rates for each block
- Calculates participation based on validator power
- Logs participation data to CSV file
- Sends Discord alerts for low participation rates (< 80%)

### 2. Signature Verification
- **Separate from participation rate monitoring** - runs independently for every block
- Verifies both valset signatures and oracle attestations
- Uses existing functions to derive EVM addresses from signatures
- Compares derived addresses against expected addresses from API
- Sends Discord alerts for any invalid signatures found

### 3. CSV Output
The monitor creates a CSV file (`vote_extension_participation.csv`) with the following columns:
- `height`: Block height
- `timestamp`: Unix timestamp
- `vote_ext_participation_rate`: Participation rate percentage
- `total_signatures`: Number of votes with extensions

## Usage

```bash
go run ./scripts/vote-ext-monitor/vote_ext_participation_rate_monitor.go \
  -rpc-url=<rpc_url> \
  -node=<node_name> \
  -swagger-api-url=<swagger_api_url> \
  -config=<config_file_path>
```

### Required Parameters
- `-rpc-url`: RPC URL of the node to monitor (default: 127.0.0.1:26657)
- `-node`: Name of the node being monitored
- `-config`: Path to the configuration file

### Optional Parameters
- `-swagger-api-url`: Swagger API URL for bridge module queries (required for signature verification)

## Configuration

Create a configuration file (e.g., `event-config.yml`) with the following structure:

```yaml
event_types:
  - alert_name: "VOTE_EXTENSION_PARTICIPATION_RATE"
    alert_type: "vote-ext-part-rate"
    query: ""
    webhook_url: "<your_discord_webhook_url>"
  - alert_name: "INVALID_SIGNATURE_ALERT"
    alert_type: "invalid-signature-alert"
    query: ""
    webhook_url: "<your_discord_webhook_url>"
  - alert_name: "LIVENESS_ALERT"
    alert_type: "liveness-alert"
    query: ""
    webhook_url: "<your_discord_webhook_url>"
  - alert_name: "CRASH_ALERT"
    alert_type: "crash-alert"
    query: ""
    webhook_url: "<your_discord_webhook_url>"
```

## Signature Verification Details

### Valset Signatures
- Verifies signatures in `valset_sigs` section of vote extensions
- Derives EVM addresses using `deriveEVMAddressFromValsetSigs()`
- Compares against expected addresses from bridge module API

### Oracle Attestations
- Verifies signatures in `oracle_attestations` section
- Derives EVM addresses using `analyzeEVMAddressesFromOracleAttestation()`
- Compares against expected addresses from bridge module API

### Discord Alerts
When invalid signatures are detected, the monitor sends a Discord alert with:
- Block height and node name
- List of all invalid signatures found
- Specific details about what failed (derived vs expected addresses, API errors, etc.)

## Output Files

- `vote_extension_participation.csv`: Participation rate data
- Console logs: Detailed information about signature verification and participation rates

## Error Handling

- Signature verification failures don't stop participation rate monitoring
- API failures for EVM address lookups are logged and included in alerts
- The monitor continues processing blocks even if some signatures can't be verified
