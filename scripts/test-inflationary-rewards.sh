#!/bin/bash

# Test script for inflationary rewards setup
echo "=== Testing Inflationary Rewards Setup ==="

# Check if we're in the right directory
if [ ! -f "./layerd" ]; then
    echo "❌ Error: layerd binary not found in current directory"
    echo "Please run this script from the directory containing the layerd binary"
    exit 1
fi
echo "✅ layerd binary found"

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "❌ Error: jq is not installed"
    echo "Please install jq first"
    exit 1
fi
echo "✅ jq is installed"

# Test RPC connection
RPC_NODE="https://node-palmito.tellorlayer.com/rpc/"
echo "Testing connection to RPC node: $RPC_NODE"

if ./layerd query block --node "$RPC_NODE" --output json > /dev/null 2>&1; then
    echo "✅ Successfully connected to RPC node"
else
    echo "❌ Error: Cannot connect to RPC node"
    echo "Please check your internet connection"
    exit 1
fi

# Test account query
TIPPER_ACCOUNT="tellor14au4s3t0z59nlkk39npcqk7d2snld5090d2pk3"
echo "Testing account query for: $TIPPER_ACCOUNT"

if ./layerd query auth account $TIPPER_ACCOUNT --node "$RPC_NODE" --output json > /dev/null 2>&1; then
    echo "✅ Successfully queried tipper account"
    
    # Get account info
    ACCOUNT_INFO=$(./layerd query auth account $TIPPER_ACCOUNT --node "$RPC_NODE" --output json)
    SEQUENCE=$(echo "$ACCOUNT_INFO" | jq -r '.account.value.sequence // .account.sequence // "0"')
    ACCOUNT_NUMBER=$(echo "$ACCOUNT_INFO" | jq -r '.account.value.account_number // .account.account_number // "0"')
    
    echo "   Sequence: $SEQUENCE"
    echo "   Account Number: $ACCOUNT_NUMBER"
else
    echo "❌ Error: Cannot query tipper account"
    echo "Please check if the account exists and is accessible"
    exit 1
fi

# Test balance query
echo "Testing balance query for tipper account"
if ./layerd query bank balances $TIPPER_ACCOUNT --node "$RPC_NODE" --output json > /dev/null 2>&1; then
    echo "✅ Successfully queried account balance"
    
    # Get balance
    BALANCE=$(./layerd query bank balances $TIPPER_ACCOUNT --node "$RPC_NODE" --output json)
    LOYA_BALANCE=$(echo "$BALANCE" | jq -r '.balances[] | select(.denom=="loya") | .amount // "0"')
    
    echo "   Loya Balance: ${LOYA_BALANCE}loya"
    
    # Check if balance is sufficient for rewards
    REQUIRED_BALANCE=700
    if [ "$LOYA_BALANCE" -ge "$REQUIRED_BALANCE" ]; then
        echo "✅ Sufficient balance for rewards (${LOYA_BALANCE}loya >= ${REQUIRED_BALANCE}loya)"
    else
        echo "⚠️  Warning: Insufficient balance for rewards (${LOYA_BALANCE}loya < ${REQUIRED_BALANCE}loya)"
        echo "   You need at least ${REQUIRED_BALANCE}loya to send rewards"
    fi
else
    echo "❌ Error: Cannot query account balance"
    exit 1
fi

# Test target accounts
TIME_REWARDS_POOL="tellor1364k288xd4r0gxlnk2rve5fm3qxamdtted7v4t"
VALIDATOR_FEE_COLLECTOR="tellor17xpfvakm2amg962yls6f84z3kell8c5ls06m3g"

echo "Testing target accounts..."
if ./layerd query auth account $TIME_REWARDS_POOL --node "$RPC_NODE" --output json > /dev/null 2>&1; then
    echo "✅ Time rewards pool account accessible"
else
    echo "❌ Error: Cannot access time rewards pool account"
fi

if ./layerd query auth account $VALIDATOR_FEE_COLLECTOR --node "$RPC_NODE" --output json > /dev/null 2>&1; then
    echo "✅ Validator fee collector account accessible"
else
    echo "❌ Error: Cannot access validator fee collector account"
fi

# Test transaction generation
echo "Testing transaction generation..."
if ./layerd tx bank send $TIPPER_ACCOUNT $TIME_REWARDS_POOL 1loya --from $TIPPER_ACCOUNT --chain-id layertest-4 --fees 20loya --generate-only > /dev/null 2>&1; then
    echo "✅ Successfully generated test transaction"
else
    echo "❌ Error: Cannot generate test transaction"
    echo "Please check your layerd configuration and keyring setup"
fi

echo ""
echo "=== Setup Test Complete ==="
echo ""
echo "If all tests passed, you can run the main script:"
echo "  ./add-manual-inflationary-rewards.sh --batch     # Test batched transaction creation"
echo "  ./add-manual-inflationary-rewards.sh --execute   # Execute a single batched transaction"
echo "  ./add-manual-inflationary-rewards.sh --continuous # Run continuously"
echo ""
echo "Make sure to run from the directory containing the layerd binary!"
