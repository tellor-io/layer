# Weekly Reports Generator

This script analyzes vote extension participation and bridge validator set update data from CSV files and generates weekly reports that are sent to Discord channels.

## Features

- **Vote Extension Analysis**: Analyzes participation rates from `vote_extension_participation.csv`
  - Average participation rate for the week
  - Number of blocks with participation below 80%
  - Lowest participation rate for the week

- **Bridge Validator Analysis**: Analyzes update timestamps from `bridge_validator_timestamps.csv`
  - Average frequency of validator set updates
  - Shortest period between updates
  - Longest period between updates

- **Scheduled Reports**: Automatically runs every Friday at 9 AM
- **Discord Integration**: Sends formatted reports to separate Discord channels

## Usage

### Basic Usage (Scheduled Mode)

```bash
go run ./scripts/weekly-reports-generator/weekly-reports-generator.go \
  -data-folder=/path/to/csv/files \
  -valset-update-channel=https://discord.com/api/webhooks/... \
  -vote-ext-participation-channel=https://discord.com/api/webhooks/...
```

### Run Once (Immediate Mode)

```bash
go run ./scripts/weekly-reports-generator/weekly-reports-generator.go \
  -data-folder=/path/to/csv/files \
  -valset-update-channel=https://discord.com/api/webhooks/... \
  -vote-ext-participation-channel=https://discord.com/api/webhooks/... \
  -run-once
```

## Parameters

- `-data-folder`: Path to the folder containing the CSV files
- `-valset-update-channel`: Discord webhook URL for bridge validator set update reports
- `-vote-ext-participation-channel`: Discord webhook URL for vote extension participation reports
- `-run-once`: (Optional) Run the report once immediately instead of waiting for Friday

## Required CSV Files

The script expects two CSV files in the specified data folder:

### vote_extension_participation.csv
```
height,timestamp,vote_ext_participation_rate
12345,1705312200,95.67
12346,1705312260,87.23
...
```

### bridge_validator_timestamps.csv
```
validator_set_update_timestamps
1705312200
1705312260
...
```

## Report Schedule

- **Day**: Every Friday
- **Time**: 9:00 AM local time
- **Period**: Last 7 days of data

## Sample Reports

### Vote Extension Participation Report
```
**Weekly Vote Extension Participation Report**

**Period:** 2024-01-08 to 2024-01-15

**Summary:**
• Total blocks analyzed: 60480
• Average participation rate: 94.23%
• Blocks below 80%: 12
• Lowest participation rate: 67.45%

**Analysis:**
This report covers the vote extension participation rates for the past week. 
Low participation rates may indicate network issues or validator problems that need attention.
```

### Bridge Validator Set Update Report
```
**Weekly Bridge Validator Set Update Report**

**Period:** 2024-01-08 to 2024-01-15

**Summary:**
• Total updates: 7
• Average frequency: 24h 0m
• Shortest period: 18h 30m
• Longest period: 32h 15m

**Analysis:**
This report covers bridge validator set update frequency for the past week. 
Regular updates are important for maintaining bridge security and performance.
```

## Dependencies

- Go 1.19+
- `github.com/tellor-io/layer/utils` package (for Discord notifications)

## Notes

- Vote extension data includes actual Unix timestamps from the CSV file
- Bridge validator timestamps are Unix timestamps (seconds since epoch)
- Reports are sent concurrently to both Discord channels
- The script will continue running indefinitely in scheduled mode until interrupted 