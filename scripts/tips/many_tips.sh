#!/bin/bash

# Stop execution if any command fails
set -e

# Configuration vars
KEYRING_BACKEND="test"
CHAIN_ID="layertest-4"
LAYER_HOME="$HOME/.layer/alice"
KEY_NAME="charlie"

# Tip amount for each query
TIP_AMOUNT="10000loya"
FEE_AMOUNT="500loya"

# Tipping style configuration - set to control which tipping approaches to run
# Options: "sequential", "parallel", "both"
TIPPING_STYLE="parallel"

echo "=== Multiple Oracle Tips Demo ==="
echo "Sending tips for multiple queries in quick succession"
echo "Tipping style: $TIPPING_STYLE"
echo ""

# Get account address
echo "Getting $KEY_NAME account address..."
user_addr=$(./layerd keys show $KEY_NAME -a --keyring-backend $KEYRING_BACKEND --home $LAYER_HOME)
echo "$KEY_NAME address: $user_addr"
echo ""

# Query and display initial account sequence
echo "Querying $KEY_NAME's initial account info..."
account_info=$(./layerd query auth account $user_addr --output json)
initial_sequence=$(echo $account_info | jq -r '.account.value.sequence')
account_number=$(echo $account_info | jq -r '.account.value.account_number')

echo "Initial sequence (nonce): $initial_sequence"
echo "Account number: $account_number"
echo ""

# Define query data from tip-queries.md
echo "Preparing query data for tips..."

# Array of query names and their corresponding query data
declare -a QUERIES=(
    "btc/usd"
    "eth/usd"
    "trb/usd"
    "saga/usd"
    "usdc/usd"
    "usdt/usd"
    "fbtc/usd"
)

declare -a QUERY_DATA=(
    "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003627463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
    "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
    "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
    "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004736167610000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
    "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004757364630000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
    "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004757364740000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
    "00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004666274630000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
)

NUM_QUERIES=${#QUERIES[@]}
echo "Number of queries to tip: $NUM_QUERIES"
echo ""

# Show what we're about to tip
echo "Queries to tip:"
for i in "${!QUERIES[@]}"; do
    echo "  $((i+1)). ${QUERIES[i]} (tip: $TIP_AMOUNT)"
done
echo ""


# Sequential tipping (if enabled)
if [ "$TIPPING_STYLE" = "sequential" ] || [ "$TIPPING_STYLE" = "both" ]; then
    echo "=== Sequential Tipping ==="
    echo "Sending tips one after another..."
    echo "Adding delay between transactions to allow commitment..."
    echo ""
    
    for i in "${!QUERIES[@]}"; do
        query_name="${QUERIES[i]}"
        query_data="${QUERY_DATA[i]}"
        note="Sequential tip #$((i+1)) for ${query_name} query"
        
        echo "Tipping query $((i+1)): $query_name"
        echo "Note: $note"
        
        result=$(./layerd tx oracle tip $query_data $TIP_AMOUNT \
            --chain-id $CHAIN_ID \
            --from $KEY_NAME \
            --fees $FEE_AMOUNT \
            --keyring-backend $KEYRING_BACKEND \
            --yes \
            --keyring-dir $LAYER_HOME \
            --note "$note" \
            --output json)
        
        txhash=$(echo $result | jq -r '.txhash')
        code=$(echo $result | jq -r '.code')
        
        echo "  Result: TxHash=$txhash, Code=$code"
        
        if [ "$code" != "0" ]; then
            echo "  Error: $(echo $result | jq -r '.raw_log')"
        fi
        
        echo ""
        
        # Add delay to allow transaction commitment before next one
        if [ $i -lt $((NUM_QUERIES-1)) ]; then
            echo "  Waiting 2 seconds for transaction commitment..."
            sleep 2
            echo ""
        fi
    done
    
    echo "=== Sequential tipping completed! ==="
    echo ""
fi

# Parallel tipping (if enabled)
if [ "$TIPPING_STYLE" = "parallel" ] || [ "$TIPPING_STYLE" = "both" ]; then
    echo "=== Parallel Tipping (Offline Signing + Rapid Broadcast) ==="
    echo "Using offline signing with pre-calculated sequence numbers"
    echo "Goal: Get multiple tips into a single block"
    echo ""
    
    # Get current sequence for parallel approach (fresh query)
    echo "Querying current account sequence for parallel approach..."
    parallel_account_info=$(./layerd query auth account $user_addr --output json)
    parallel_sequence=$(echo $parallel_account_info | jq -r '.account.value.sequence')
    
    echo "Current sequence for parallel approach: $parallel_sequence"
    echo ""
    
    # Clean up any existing transaction files
    rm -f tip_tx_*.json
    
    # Generate and sign multiple tip transactions with sequential sequence numbers
    echo "Generating and signing $NUM_QUERIES tip transactions with sequential sequences..."
    for i in "${!QUERIES[@]}"; do
        query_name="${QUERIES[i]}"
        query_data="${QUERY_DATA[i]}"
        sequence=$((parallel_sequence + i))
        note="Parallel tip #$((i+1)) for ${query_name} query (seq: $sequence)"
        
        echo "Generating tip tx $((i+1)): $query_name with sequence $sequence"
        
        # Generate unsigned transaction
        ./layerd tx oracle tip $query_data $TIP_AMOUNT \
            --from $user_addr \
            --fees $FEE_AMOUNT \
            --keyring-backend $KEYRING_BACKEND \
            --keyring-dir $LAYER_HOME \
            --sequence $sequence \
            --offline \
            --account-number $account_number \
            --note "$note" \
            --generate-only > tip_tx_${i}.json
        
        # Sign the transaction offline
        ./layerd tx sign \
            --from $user_addr \
            tip_tx_${i}.json \
            --chain-id=$CHAIN_ID \
            --keyring-backend $KEYRING_BACKEND \
            --keyring-dir $LAYER_HOME \
            --offline \
            --account-number $account_number \
            --sequence $sequence \
            --yes > tip_tx_${i}_signed.json
    done
    
    echo ""
    echo "All tip transactions generated and signed!"
    echo ""
    
    # Broadcast all transactions rapidly in parallel
    echo "Broadcasting all tip transactions rapidly in parallel..."
    echo "Transaction hashes:"
    
    for i in "${!QUERIES[@]}"; do
        query_name="${QUERIES[i]}"
        echo -n "TIP $((i+1)) ($query_name): "
        ./layerd tx broadcast tip_tx_${i}_signed.json #--output json | jq -r '.txhash' &
    done
    
    # Wait for all background broadcasts to complete
    wait
    
    echo "=== Parallel tipping completed! ==="
    echo ""
fi

# Clean up temporary files
rm -f tip_result_*.json tip_tx_*.json
