#!/bin/bash


# Stop execution if any command fails
set -e

# Configuration variables (matching the start_two_chains.sh setup)
KEYRING_BACKEND="test"
CHAIN_ID="layertest-4"
BILL_HOME="$HOME/.layer/bill"
ALICE_HOME="$HOME/.layer/alice"

# Number of transactions to send in one block
NUM_TXS=3

echo "=== Multi-Transaction Demo ==="
echo "Attempting to send $NUM_TXS transactions from bill to alice in one block"
echo ""

# Get account addresses
echo "Getting account addresses..."
bill=$(./layerd keys show bill -a --keyring-backend $KEYRING_BACKEND --home $BILL_HOME)
alice=$(./layerd keys show alice -a --keyring-backend $KEYRING_BACKEND --home $ALICE_HOME)

echo "Bill address: $bill"
echo "Alice address: $alice"
echo ""

# Query current account info to get sequence and account number
echo "Querying bill's account info..."
account_info=$(./layerd query auth account $bill --output json)
current_sequence=$(echo $account_info | jq -r '.account.value.sequence')
account_number=$(echo $account_info | jq -r '.account.value.account_number')

echo "Current sequence: $current_sequence"
echo "Account number: $account_number"

# Validate that we got valid values
if [ "$current_sequence" = "null" ] || [ "$account_number" = "null" ] || [ -z "$current_sequence" ] || [ -z "$account_number" ]; then
    echo "Error: Failed to get valid account info. Check that the chain is running and bill account exists."
    echo "Raw account info:"
    echo $account_info
    exit 1
fi

echo ""

# Clean up any existing transaction files
rm -f tx_*.json

# Generate multiple transactions with sequential sequence numbers
echo "Generating $NUM_TXS transactions..."
for i in $(seq 0 $((NUM_TXS-1))); do
    sequence=$((current_sequence + i))
    amount=$((i + 1))  # send 1loya, 2loya, 3loya, etc.
    
    echo "Generating tx $((i+1)): sending ${amount}loya with sequence $sequence"
    
    ./layerd tx bank send $bill $alice ${amount}loya \
        --from $bill \
        --fees 500loya \
        --keyring-backend $KEYRING_BACKEND \
        --keyring-dir $BILL_HOME \
        --sequence $sequence \
        --offline \
        --account-number $account_number \
        --generate-only > tx_${i}.json
    
    ./layerd tx sign \
        --from $bill \
        tx_${i}.json \
        --chain-id=$CHAIN_ID \
        --keyring-backend $KEYRING_BACKEND \
        --keyring-dir $BILL_HOME \
        --offline \
        --account-number $account_number \
        --sequence $sequence \
        --yes > tx_${i}_signed.json
done

echo ""

echo ""

# Broadcast all transactions as quickly as possible
echo "Broadcasting all transactions rapidly..."
echo "Transaction hashes:"

for i in $(seq 0 $((NUM_TXS-1))); do
    echo -n "TX $((i+1)): "
    ./layerd tx broadcast tx_${i}_signed.json #--output json | jq -r '.txhash' &
done

# Wait for all background broadcasts to complete
wait

echo ""
echo "All transactions broadcasted!"
echo ""
echo "Wait a moment for the next block, then check if they were included..."
sleep 3

echo ""
echo "Checking recent transactions for bill account..."
./layerd query txs --query "message.sender='$bill'" --limit $NUM_TXS --order_by "desc"

echo ""
echo "=== Demo Complete ==="
