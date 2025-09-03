# Inflationary Rewards Script

This script creates and executes batched Cosmos SDK transactions for sending inflationary rewards to the time-based rewards pool and validator fee collector pool.

## Overview

The script sends a total of 700 loya per block, distributed as follows:
- **75% (525 loya)** to the time-based rewards pool
- **25% (175 loya)** to the validator fee collector pool

## Prerequisites

1. **jq**: JSON processor for manipulating transaction files
   ```bash
   # macOS
   brew install jq
   
   # Ubuntu/Debian
   sudo apt-get install jq
   
   # CentOS/RHEL
   sudo yum install jq
   ```

2. **layerd**: The Layer blockchain CLI binary must be in the same directory as the script

3. **Account Setup**: Your tipper account must have sufficient loya to send rewards

4. **Internet Connection**: Access to the Tellor RPC node

## Configuration

Before running the script, update these variables in the script:

```bash
export TIPPER_ACCOUNT="your_tipper_account_address"
export TIME_REWARDS_POOL="time_based_rewards_pool_address"
export VALIDATOR_FEE_COLLECTOR="validator_fee_collector_address"
```

## Testing Your Setup

Before running the main script, test your setup:

```bash
# Make the test script executable
chmod +x test-inflationary-rewards.sh

# Run the test script
./test-inflationary-rewards.sh
```

This will verify:
- ✅ layerd binary is accessible
- ✅ jq is installed
- ✅ RPC connection works
- ✅ Account queries work
- ✅ Sufficient balance for rewards
- ✅ Target accounts are accessible
- ✅ Transaction generation works

## Usage

```bash
./add-manual-inflationary-rewards.sh --batch
```

This creates the following files:
- `tx_batch_time_rewards.json` - Individual time rewards transaction
- `tx_batch_validator_fees.json` - Individual validator fees transaction  
- `tx_batch_final.json` - Combined batched transaction with both messages

### 2. Execute Batched Transaction Immediately

```bash
./add-manual-inflationary-rewards.sh --execute
```

This creates, signs, and broadcasts the batched transaction for the current block.

### 3. Run Continuously (Every Block)

```bash
./add-manual-inflationary-rewards.sh --continuous
```

This runs indefinitely, sending rewards every block.

### 4. Run for Specific Number of Blocks

```bash
./add-manual-inflationary-rewards.sh --num-blocks 10
```

This runs for 10 blocks then exits.

### 5. Show Help

```bash
./add-manual-inflationary-rewards.sh --help
```

## How Batched Transactions Work

The script creates a proper batched transaction by:

1. **Querying** the tipper account's current sequence and account number from the blockchain
2. **Generating** two separate bank send transactions
3. **Extracting** the message from the second transaction
4. **Combining** both messages into the first transaction's messages array
5. **Signing** the combined transaction
6. **Broadcasting** with proper sequence and account number flags
7. **Re-querying** the blockchain for fresh sequence numbers on each new transaction

The resulting transaction has this structure:
```json
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.bank.v1beta1.MsgSend",
        "from_address": "tipper_address",
        "to_address": "time_rewards_pool",
        "amount": [{"denom": "loya", "amount": "525"}]
      },
      {
        "@type": "/cosmos.bank.v1beta1.MsgSend", 
        "from_address": "tipper_address",
        "to_address": "validator_fee_collector",
        "amount": [{"denom": "loya", "amount": "175"}]
      }
    ]
  }
}
```

## Logging

The script logs all activities to `/var/log/inflationary-rewards.log` and also displays them on the console.

## Error Handling

- **Retry Logic**: Failed transactions are retried up to 5 times
- **Block Monitoring**: The script waits for each new block before proceeding
- **Graceful Failures**: Errors are logged and the script continues to the next block
- **Sequence Management**: Automatically queries the blockchain for fresh sequence numbers for each transaction

## Security Notes

- Ensure your tipper account has sufficient funds
- The script uses the `test` keyring backend by default
- Update the chain ID if running on a different network
- Consider running in a controlled environment first

## Example Output

```
[2024-01-15 10:30:00] Starting inflationary rewards script
[2024-01-15 10:30:00] Configuration:
[2024-01-15 10:30:00]   Chain ID: tellor-1
[2024-01-15 10:30:00]   Tipper Account: tellor1zhg69su8p5zplr7jkzkav7874ekncn83592rlk
[2024-01-15 10:30:00]   Time Rewards Pool: tellor1364k288xd4r0gxlnk2rve5fm3qxamdtted7v4t
[2024-01-15 10:30:00]   Validator Fee Collector: tellor17xpfvakm2amg962yls6f84z3kell8c5ls06m3g
[2024-01-15 10:30:00]   Total Rewards per Block: 700loya
[2024-01-15 10:30:00]   Time Rewards: 525loya (75%)
[2024-01-15 10:30:00]   Validator Fees: 175loya (25%)
[2024-01-15 10:30:00] Creating batched transaction for block 12345
[2024-01-15 10:30:00] Batched transaction created: tx_batch_final.json
[2024-01-15 10:30:00] This transaction contains both:
[2024-01-15 10:30:00]   - 525loya to time rewards pool
[2024-01-15 10:30:00]   - 175loya to validator fee collector
[2024-01-15 10:30:00] Total fees: 40loya
```

## Troubleshooting

### Common Issues

1. **jq not found**: Install jq as described in prerequisites
2. **Insufficient funds**: Ensure tipper account has enough loya
3. **Invalid addresses**: Double-check all account addresses
4. **Chain not running**: Ensure layerd is accessible and chain is running
5. **Sequence errors**: If you get sequence errors, the script automatically handles this, but you can manually refresh account info if needed

### Debug Mode

To see more detailed output, you can modify the script to increase verbosity or check the log file at `/var/log/inflationary-rewards.log`.
