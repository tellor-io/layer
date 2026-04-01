#!/bin/bash

# | Test Name | Query Type | Query Details | Expected Outcome |
# |-----------|------------|---------------|------------------|
# | EVMCALL GOOD | EVMCall | Chain 1, contract 0x5589e306b1920f009979a50b88cae32aecd471e4, calldata 0x7f629c65 | Good value (0x1baf80 = 1814400) |
# | CYCLE LIST BAD (eth/usd) | SpotPrice | eth/usd | Bad value (using new reporter account) |
# | IN TELLIOT, NOT CONFIG GOOD (sfrxusd/usd) | SpotPrice | sfrxusd/usd | Good value |
# | IN TELLIOT, NOT CONFIG BAD (ltc/usd) | SpotPrice | ltc/usd | Bad value (using new reporter account) |
# | IN CONFIG, NOT TELLIOT (soup/usd) | SpotPrice | soup/usd | Test value |
# | TRBBRIDGE V1 FAILS | TRBBridge | Deposit ID 142 | Legacy query type is rejected |
# | TRBBRIDGEV2 SUCCEEDS | TRBBridgeV2 | Deposit ID 142 | Bridge deposit report succeeds |
# | NFLWINNER (not in telliot or config) | NFLWinner | Week 9, colts vs steelers | Winner: steelers |
# | EVMCALL BAD | EVMCall | Chain 1, contract 0x5589e306b1920f009979a50b88cae32aecd471e4, calldata 0x7f629c65 | Bad value (0x2bad00 = 2862336) |

set +e  # Don't exit on errors, we want to track them

export HOME_DIR="$HOME/.layer-chains/testnet/layer"

# Arrays to track test results
declare -a TEST_NAMES
declare -a TEST_RESULTS

TX_QUERY_RESULT=""
LAST_TX_CODE=""
LAST_TX_RAW_LOG=""
LAST_TX_GAS_USED=""

extract_txhash() {
  echo "$1" | awk '/^txhash: / {print $2; exit}'
}

query_tx_with_retry() {
  local txhash=$1
  local description=$2
  local max_attempts=${3:-12}
  local sleep_seconds=${4:-1}
  local attempt=1

  TX_QUERY_RESULT=""

  while [ $attempt -le $max_attempts ]; do
    TX_QUERY_RESULT=$(./layerd query tx "$txhash" --home $HOME_DIR/alice --output json 2>&1)
    if [ $? -eq 0 ]; then
      return 0
    fi

    if [ $attempt -eq 1 ]; then
      echo "⚠️  $description tx not found immediately, waiting ${sleep_seconds}s and retrying..."
    elif [ $attempt -lt $max_attempts ]; then
      echo "⚠️  $description tx still pending (attempt $attempt/$max_attempts), waiting ${sleep_seconds}s..."
    fi

    if [ $attempt -lt $max_attempts ]; then
      sleep "$sleep_seconds"
    fi

    attempt=$((attempt + 1))
  done

  return 1
}

confirm_tx_status() {
  local txhash=$1
  local description=$2
  local max_attempts=${3:-12}
  local sleep_seconds=${4:-1}
  local tx_code
  local gas_used
  local raw_log

  if ! query_tx_with_retry "$txhash" "$description" "$max_attempts" "$sleep_seconds"; then
    LAST_TX_CODE=""
    LAST_TX_RAW_LOG="$TX_QUERY_RESULT"
    LAST_TX_GAS_USED=""
    echo "❌ ERROR: $description tx not found after $max_attempts attempts"
    echo "Last error: $TX_QUERY_RESULT"
    return 1
  fi

  tx_code=$(echo "$TX_QUERY_RESULT" | jq -r '.code // 0')
  gas_used=$(echo "$TX_QUERY_RESULT" | jq -r '.gas_used // "unknown"')
  raw_log=$(echo "$TX_QUERY_RESULT" | jq -r '.raw_log // empty')
  LAST_TX_CODE="$tx_code"
  LAST_TX_RAW_LOG="$raw_log"
  LAST_TX_GAS_USED="$gas_used"

  if [ "$tx_code" = "0" ]; then
    echo "✓ $description tx confirmed (code: 0, gas used: ${gas_used:-unknown})"
    return 0
  fi

  echo "⚠️  $description tx returned code: $tx_code (gas used: ${gas_used:-unknown})"
  if [ -n "$raw_log" ]; then
    echo "  raw_log: $raw_log"
  fi
  return 1
}

retry_submit_after_retip() {
  local query_data=$1
  local tip_amount=$2
  local submit_value=$3
  local submit_gas_limit=$4
  local from_account=${5:-"alice"}
  local retry_tip_output
  local retry_tip_txhash
  local retry_submit_output
  local retry_submit_txhash

  echo "Re-submitting tip to reopen the reporting window..."
  retry_tip_output=$(./layerd tx oracle tip "$query_data" "$tip_amount" \
    --from $from_account \
    --chain-id layer-local-1 \
    --keyring-backend test \
    --home $HOME_DIR/alice \
    --keyring-dir $HOME_DIR/alice \
    --fees 500loya \
    --unordered \
    --timeout-duration 30s \
    --yes 2>&1)

  retry_tip_txhash=$(extract_txhash "$retry_tip_output")
  if [ -z "$retry_tip_txhash" ]; then
    echo "❌ ERROR: Could not extract retry tip tx hash"
    echo "Full output:"
    echo "$retry_tip_output"
    return 1
  fi

  echo "✓ Retry tip tx hash extracted: $retry_tip_txhash"
  echo "Checking retry tip tx status..."
  if ! confirm_tx_status "$retry_tip_txhash" "Retry tip" 12 1; then
    return 1
  fi

  if [ -n "$submit_gas_limit" ]; then
    echo "Retrying submit value after reopening the reporting window (custom gas limit: $submit_gas_limit)..."
    retry_submit_output=$(./layerd tx oracle submit-value "$query_data" "$submit_value" \
      --from $from_account \
      --chain-id layer-local-1 \
      --keyring-backend test \
      --home $HOME_DIR/alice \
      --keyring-dir $HOME_DIR/alice \
      --fees 650loya \
      --gas $submit_gas_limit \
      --unordered \
      --timeout-duration 30s \
      --yes 2>&1)
  else
    echo "Retrying submit value after reopening the reporting window (using auto gas estimation)..."
    retry_submit_output=$(./layerd tx oracle submit-value "$query_data" "$submit_value" \
      --from $from_account \
      --chain-id layer-local-1 \
      --keyring-backend test \
      --home $HOME_DIR/alice \
      --keyring-dir $HOME_DIR/alice \
      --fees 650loya \
      --unordered \
      --timeout-duration 30s \
      --yes 2>&1)
  fi

  echo "--- Full retry submit-value command output ---"
  echo "$retry_submit_output"
  echo "--- End of output ---"

  retry_submit_txhash=$(extract_txhash "$retry_submit_output")
  if [ -z "$retry_submit_txhash" ]; then
    echo "❌ ERROR: Could not extract retry submit tx hash"
    echo "Full output:"
    echo "$retry_submit_output"
    return 1
  fi

  echo "✓ Retry submit tx hash extracted: $retry_submit_txhash"
  echo "Checking retry submit tx status..."
  if ! confirm_tx_status "$retry_submit_txhash" "Retry submit" 12 1; then
    return 1
  fi

  return 0
}

get_report_count() {
  local query_id=$1

  ./layerd query oracle get-reportsby-qid "$query_id" \
    --home $HOME_DIR/alice \
    --output json \
    --page-limit 1 \
    --page-count-total 2>/dev/null | jq -r '.pagination.total // "0"'
}

ensure_trbbridgev2_spec_registered() {
  echo "Checking if TRBBridgeV2 data spec is registered..."
  if ./layerd query registry data-spec TRBBridgeV2 --home $HOME_DIR/alice &>/dev/null; then
    echo "TRBBridgeV2 data spec found."
    echo ""
    return 0
  fi

  echo "⚠️  WARNING: TRBBridgeV2 data spec is not registered on this chain."
  echo "Restart the local chain with the updated start_a_chain.sh to include it in genesis."
  echo ""
  return 1
}

submit_and_expect_failure() {
  local description=$1
  local query_data=$2
  local submit_value=$3
  local expected_error=$4
  local submit_gas_limit=${5:-""}
  local from_account=${6:-"alice"}
  local test_status="SUCCESS"
  local submit_output
  local submit_txhash

  TEST_NAMES+=("$description")

  echo "=========================================="
  echo "Testing: $description"
  echo "=========================================="

  if [ -n "$submit_gas_limit" ]; then
    echo "Submitting value (expecting failure, custom gas limit: $submit_gas_limit)..."
    submit_output=$(./layerd tx oracle submit-value "$query_data" "$submit_value" \
      --from $from_account \
      --chain-id layer-local-1 \
      --keyring-backend test \
      --home $HOME_DIR/alice \
      --keyring-dir $HOME_DIR/alice \
      --fees 650loya \
      --gas $submit_gas_limit \
      --unordered \
      --timeout-duration 30s \
      --yes 2>&1)
  else
    echo "Submitting value (expecting failure, using auto gas estimation)..."
    submit_output=$(./layerd tx oracle submit-value "$query_data" "$submit_value" \
      --from $from_account \
      --chain-id layer-local-1 \
      --keyring-backend test \
      --home $HOME_DIR/alice \
      --keyring-dir $HOME_DIR/alice \
      --fees 650loya \
      --unordered \
      --timeout-duration 30s \
      --yes 2>&1)
  fi

  echo "  Query data length: ${#query_data} chars"
  echo "  Value length: ${#submit_value} chars"
  echo "--- Full expected-failure submit-value command output ---"
  echo "$submit_output"
  echo "--- End of output ---"

  submit_txhash=$(extract_txhash "$submit_output")
  if [ -z "$submit_txhash" ]; then
    echo "❌ ERROR: Could not extract expected-failure submit tx hash"
    echo "Full output:"
    echo "$submit_output"
    test_status="FAILED"
    TEST_RESULTS+=("$test_status")
    echo ""
    return 1
  fi

  echo "✓ Expected-failure submit tx hash extracted: $submit_txhash"
  echo "Checking expected-failure submit tx status..."
  if confirm_tx_status "$submit_txhash" "Expected-failure submit" 12 1; then
    echo "❌ ERROR: Submit tx unexpectedly succeeded"
    test_status="FAILED"
  elif [ -z "$LAST_TX_CODE" ]; then
    echo "❌ ERROR: Submit tx status could not be confirmed"
    test_status="FAILED"
  elif echo "$LAST_TX_RAW_LOG" | grep -Fqi "$expected_error"; then
    echo "✅ SUCCESS: Submit failed as expected"
  else
    echo "❌ ERROR: Submit failed, but not with the expected error"
    echo "Expected to find: $expected_error"
    test_status="FAILED"
  fi

  TEST_RESULTS+=("$test_status")
  echo ""
}

# Helper function to submit tip, extract txhash, wait, and check status
submit_and_check() {
  local description=$1
  local query_data=$2
  local tip_amount=$3
  local submit_value=$4
  local query_id=$5
  local submit_gas_limit=${6:-""}  # Gas limit for submit-value only (tips always use auto)
  local from_account=${7:-"alice"}  # Optional: account to use (defaults to alice)
  local test_status="SUCCESS"
  local TIP_TX_SUCCESS=true  # Default to true if no tip is required
  
  TEST_NAMES+=("$description")
  
  echo "=========================================="
  echo "Testing: $description"
  echo "=========================================="
  
  if [ -n "$tip_amount" ]; then
    echo "Submitting tip (using auto gas estimation)..."
    echo "  Query data length: ${#query_data} chars"
    echo "  Tip amount: $tip_amount"
    
    # Tips always use auto gas estimation with unordered mode
    TIP_OUTPUT=$(./layerd tx oracle tip "$query_data" "$tip_amount" \
      --from $from_account \
      --chain-id layer-local-1 \
      --keyring-backend test \
      --home $HOME_DIR/alice \
      --keyring-dir $HOME_DIR/alice \
      --fees 500loya \
      --unordered \
      --timeout-duration 30s \
      --yes 2>&1)
    
    TIP_TXHASH=$(extract_txhash "$TIP_OUTPUT")
    
    if [ -z "$TIP_TXHASH" ]; then
      echo "⚠️  Tip tx hash not found in first attempt, checking for errors..."
      if echo "$TIP_OUTPUT" | grep -qi "error\|failed\|rejected"; then
        echo "❌ ERROR found in output:"
        echo "$TIP_OUTPUT" | grep -i "error\|failed\|rejected" | head -5
      fi
      echo "Retrying hash extraction in 1s..."
      sleep 1
      TIP_TXHASH=$(extract_txhash "$TIP_OUTPUT")
    fi
    
    if [ -z "$TIP_TXHASH" ]; then
      echo "❌ ERROR: Could not extract tip tx hash"
      echo "Full output:"
      echo "$TIP_OUTPUT"
      test_status="FAILED"
      TEST_RESULTS+=("$test_status")
      echo ""
      return 1
    fi
    
    echo "✓ Tip tx hash extracted: $TIP_TXHASH"
    sleep 1
    
    echo "Checking tip tx status..."
    TIP_TX_SUCCESS=false
    if confirm_tx_status "$TIP_TXHASH" "Tip" 12 1; then
      TIP_TX_SUCCESS=true
    else
      test_status="FAILED"
      TEST_RESULTS+=("$test_status")
      echo ""
      return 1
    fi
  fi
  
  if [ -n "$submit_value" ]; then
    # Only proceed if tip tx succeeded (or if no tip was required)
    if [ -n "$tip_amount" ] && [ "$TIP_TX_SUCCESS" != "true" ]; then
      echo "⚠️  Skipping submit-value because tip tx failed"
      test_status="FAILED"
      TEST_RESULTS+=("$test_status")
      echo ""
      return 1
    fi
    # Get initial report count before submitting
    if [ -n "$query_id" ]; then
      echo "Querying initial report count for query ID: $query_id"
      INITIAL_COUNT=$(get_report_count "$query_id")
      echo "Initial report count: $INITIAL_COUNT"
    fi
    
    # Try to submit value with retries
    SUBMIT_ATTEMPT=1
    SUBMIT_TXHASH=""
    MAX_SUBMIT_ATTEMPTS=3
    
    while [ $SUBMIT_ATTEMPT -le $MAX_SUBMIT_ATTEMPTS ]; do
      if [ $SUBMIT_ATTEMPT -eq 1 ]; then
        if [ -n "$submit_gas_limit" ]; then
          echo "Submitting value (custom gas limit: $submit_gas_limit)..."
        else
          echo "Submitting value (using auto gas estimation)..."
        fi
      else
        if [ -n "$submit_gas_limit" ]; then
          echo "Retrying submit value (attempt $SUBMIT_ATTEMPT/$MAX_SUBMIT_ATTEMPTS, custom gas limit: $submit_gas_limit)..."
        else
          echo "Retrying submit value (attempt $SUBMIT_ATTEMPT/$MAX_SUBMIT_ATTEMPTS, using auto gas estimation)..."
        fi
      fi
      
      echo "  Query data length: ${#query_data} chars"
      echo "  Value length: ${#submit_value} chars"
      
      if [ -n "$submit_gas_limit" ]; then
        SUBMIT_OUTPUT=$(./layerd tx oracle submit-value "$query_data" "$submit_value" \
          --from $from_account \
          --chain-id layer-local-1 \
          --keyring-backend test \
          --home $HOME_DIR/alice \
          --keyring-dir $HOME_DIR/alice \
          --fees 650loya \
          --gas $submit_gas_limit \
          --unordered \
          --timeout-duration 30s \
          --yes 2>&1)
      else
        SUBMIT_OUTPUT=$(./layerd tx oracle submit-value "$query_data" "$submit_value" \
          --from $from_account \
          --chain-id layer-local-1 \
          --keyring-backend test \
          --home $HOME_DIR/alice \
          --keyring-dir $HOME_DIR/alice \
          --fees 650loya \
          --unordered \
          --timeout-duration 30s \
          --yes 2>&1)
      fi
      
      # Log the full output for debugging
      echo "--- Full submit-value command output (attempt $SUBMIT_ATTEMPT) ---"
      echo "$SUBMIT_OUTPUT"
      echo "--- End of output ---"
      
      SUBMIT_TXHASH=$(extract_txhash "$SUBMIT_OUTPUT")
      
      if [ -n "$SUBMIT_TXHASH" ]; then
        echo "✓ Submit tx hash extracted: $SUBMIT_TXHASH"
        break
      fi
      
      # Check for errors in output
      if echo "$SUBMIT_OUTPUT" | grep -qi "error\|failed\|rejected"; then
        echo "⚠️  ERROR found in output:"
        echo "$SUBMIT_OUTPUT" | grep -i "error\|failed\|rejected" | head -10
      fi
      
      if [ $SUBMIT_ATTEMPT -lt $MAX_SUBMIT_ATTEMPTS ]; then
        echo "⚠️  Submit tx hash not found, waiting 0.5s before retry..."
        sleep 0.5
      fi
      
      SUBMIT_ATTEMPT=$((SUBMIT_ATTEMPT + 1))
    done
    
    if [ -z "$SUBMIT_TXHASH" ]; then
      echo "❌ ERROR: Could not extract submit tx hash after $MAX_SUBMIT_ATTEMPTS attempts"
      echo "Full output from last attempt:"
      echo "$SUBMIT_OUTPUT"
      test_status="FAILED"
      TEST_RESULTS+=("$test_status")
      echo ""
      return 1
    fi
    echo "Checking submit tx status..."
    if ! confirm_tx_status "$SUBMIT_TXHASH" "Submit" 12 1; then
      if [ "$LAST_TX_CODE" = "1114" ] && [ -n "$tip_amount" ]; then
        echo "⚠️  Submission window expired before the tx landed. Re-tipping and retrying submit once..."
        if ! retry_submit_after_retip "$query_data" "$tip_amount" "$submit_value" "$submit_gas_limit" "$from_account"; then
          test_status="FAILED"
          TEST_RESULTS+=("$test_status")
          echo ""
          return 1
        fi
      else
        test_status="FAILED"
        TEST_RESULTS+=("$test_status")
        echo ""
        return 1
      fi
    fi
    
    # Get final report count and verify it increased by 1
    if [ -n "$query_id" ]; then
      echo ""
      echo "Querying final report count for query ID: $query_id"
      FINAL_COUNT=$(get_report_count "$query_id")
      echo "Final report count: $FINAL_COUNT"
      
      EXPECTED_COUNT=$((INITIAL_COUNT + 1))
      if [ "$FINAL_COUNT" -eq "$EXPECTED_COUNT" ]; then
        echo "✅ SUCCESS: Report count increased by 1 (from $INITIAL_COUNT to $FINAL_COUNT)"
      else
        echo "Count did not increase yet, waiting 2s and retrying..."
        sleep 2
        FINAL_COUNT=$(get_report_count "$query_id")
        echo "Final report count (after retry): $FINAL_COUNT"
        
        if [ "$FINAL_COUNT" -eq "$EXPECTED_COUNT" ]; then
          echo "✅ SUCCESS: Report count increased by 1 (from $INITIAL_COUNT to $FINAL_COUNT)"
        else
          echo "❌ ERROR: Expected report count to be $EXPECTED_COUNT but got $FINAL_COUNT"
          test_status="FAILED"
        fi
      fi
    fi
  elif [ -n "$query_id" ]; then
    # If no submit_value but there's a query_id, just query once
    echo "Querying reports by query ID: $query_id"
    REPORT_COUNT=$(get_report_count "$query_id")
    echo "Total reports: $REPORT_COUNT"
  fi
  
  TEST_RESULTS+=("$test_status")
  echo ""
}

# Helper function to create and setup a reporter
create_reporter_account() {
  local reporter_name=$1
  local description=$2
  
  echo "=========================================="
  echo "Creating new account for $description..."
  echo "=========================================="
  echo "Account name: $reporter_name"
  
  # Create the account (suppress mnemonic output for cleaner logs)
  ./layerd keys add "$reporter_name" \
    --keyring-backend test \
    --home $HOME_DIR/alice \
    --keyring-dir $HOME_DIR/alice \
    --output json > /tmp/new_account_${reporter_name}.json 2>&1
  
  # Extract the address
  local reporter_addr=$(cat /tmp/new_account_${reporter_name}.json | jq -r '.address')
  echo "✓ New account created!"
  echo "  Address: $reporter_addr"
  echo "  Name: $reporter_name"
  
  # Fund the new account from alice with 1.2M loya
  echo ""
  echo "Funding new account with 1200000loya from alice..."
  
  FUND_OUTPUT=$(./layerd tx bank send alice "$reporter_addr" 1200000loya \
    --from alice \
    --chain-id layer-local-1 \
    --keyring-backend test \
    --home $HOME_DIR/alice \
    --keyring-dir $HOME_DIR/alice \
    --fees 500loya \
    --unordered \
    --timeout-duration 30s \
    --yes 2>&1)
  
  FUND_TXHASH=$(extract_txhash "$FUND_OUTPUT")
  if [ -n "$FUND_TXHASH" ]; then
    echo "✓ Funding tx sent: $FUND_TXHASH"
    sleep 2
    echo "✓ Account funded with 1200000loya"
  else
    echo "⚠️  Warning: Could not extract funding tx hash, but proceeding anyway..."
  fi
  
  # Get alice's validator address
  echo ""
  echo "Getting alice's validator address..."
  ALICE_VALOPER=$(./layerd keys show alice --bech val --keyring-backend test --home $HOME_DIR/alice --keyring-dir $HOME_DIR/alice 2>&1 | grep -o 'tellorvaloper[a-z0-9]*')
  if [ -z "$ALICE_VALOPER" ]; then
    echo "⚠️  Could not find validator address, trying alternative method..."
    ALICE_VALOPER=$(./layerd query staking validators --home $HOME_DIR/alice --output json 2>/dev/null | jq -r '.validators[0].operator_address')
  fi
  echo "✓ Alice's validator: $ALICE_VALOPER"
  
  # Delegate 1M loya to alice's validator
  echo ""
  echo "Delegating 1000000loya to alice's validator..."
  sleep 1
  
  DELEGATE_ATTEMPT=1
  DELEGATE_TXHASH=""
  while [ $DELEGATE_ATTEMPT -le 2 ]; do
    if [ $DELEGATE_ATTEMPT -eq 2 ]; then
      echo "Retrying delegation (attempt 2/2)..."
      sleep 1
    fi
    
    DELEGATE_OUTPUT=$(./layerd tx staking delegate "$ALICE_VALOPER" 1000000loya \
      --from "$reporter_name" \
      --chain-id layer-local-1 \
      --keyring-backend test \
      --home $HOME_DIR/alice \
      --keyring-dir $HOME_DIR/alice \
      --fees 500loya \
      --unordered \
      --timeout-duration 30s \
      --yes 2>&1)
  
    DELEGATE_TXHASH=$(extract_txhash "$DELEGATE_OUTPUT")
    
    if [ -n "$DELEGATE_TXHASH" ]; then
      echo "✓ Delegation tx sent: $DELEGATE_TXHASH"
      sleep 2
      echo "✓ Delegated 1000000loya to validator"
      break
    fi
    
    if [ $DELEGATE_ATTEMPT -eq 1 ]; then
      echo "⚠️  Could not extract delegation tx hash, will retry..."
    else
      echo "⚠️  Warning: Could not extract delegation tx hash after 2 attempts, but proceeding anyway..."
    fi
    
    DELEGATE_ATTEMPT=$((DELEGATE_ATTEMPT + 1))
  done
  
  # Create reporter for the new account
  echo ""
  echo "Creating reporter for new account..."
  echo "Command: ./layerd tx reporter create-reporter 0.1 1000000 \"$reporter_name\" --from \"$reporter_name\" ..."
  sleep 1
  
  CREATE_REPORTER_ATTEMPT=1
  CREATE_REPORTER_TXHASH=""
  while [ $CREATE_REPORTER_ATTEMPT -le 2 ]; do
    if [ $CREATE_REPORTER_ATTEMPT -eq 2 ]; then
      echo "Retrying create-reporter (attempt 2/2)..."
      sleep 1
    fi
    
    CREATE_REPORTER_OUTPUT=$(./layerd tx reporter create-reporter 0.1 1000000 "$reporter_name" \
      --from "$reporter_name" \
      --chain-id layer-local-1 \
      --keyring-backend test \
      --home $HOME_DIR/alice \
      --keyring-dir $HOME_DIR/alice \
      --fees 500loya \
      --unordered \
      --timeout-duration 30s \
      --yes 2>&1)
  
    # Log the full output for debugging
    echo "--- Full create-reporter command output (attempt $CREATE_REPORTER_ATTEMPT) ---"
    echo "$CREATE_REPORTER_OUTPUT"
    echo "--- End of output ---"
  
    CREATE_REPORTER_TXHASH=$(extract_txhash "$CREATE_REPORTER_OUTPUT")
    
    if [ -n "$CREATE_REPORTER_TXHASH" ]; then
      echo "✓ Create reporter tx sent: $CREATE_REPORTER_TXHASH"
      sleep 2
      
      # Verify the transaction status
      echo "Checking create-reporter tx status..."
      if confirm_tx_status "$CREATE_REPORTER_TXHASH" "Create reporter" 12 1; then
        echo "✓ Reporter created successfully"
      else
        echo "⚠️  Reporter creation failed, later submissions with $reporter_name may also fail."
      fi
      break
    fi
    
    # Check for errors in output
    if echo "$CREATE_REPORTER_OUTPUT" | grep -qi "error\|failed\|rejected"; then
      echo "⚠️  ERROR found in create-reporter output:"
      echo "$CREATE_REPORTER_OUTPUT" | grep -i "error\|failed\|rejected" | head -10
    fi
    
    if [ $CREATE_REPORTER_ATTEMPT -eq 1 ]; then
      echo "⚠️  Could not extract create-reporter tx hash, will retry..."
    else
      echo "⚠️  Warning: Could not extract create-reporter tx hash after 2 attempts, but proceeding anyway..."
      echo "Full output from last attempt was shown above."
    fi
    
    CREATE_REPORTER_ATTEMPT=$((CREATE_REPORTER_ATTEMPT + 1))
  done
  
  # Clean up temp file
  rm -f /tmp/new_account_${reporter_name}.json
  echo ""
}

# Check if EVMCall data spec is registered and run EVMCALL GOOD first
echo "Checking if EVMCall data spec is registered..."
if ./layerd query registry data-spec EVMCall --home $HOME_DIR/alice &>/dev/null; then
  echo "EVMCall data spec found, proceeding with EVMCALL GOOD test..."
  
  # EVMCALL GOOD (run first)
  EVMCALL_QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000047f629c6500000000000000000000000000000000000000000000000000000000"
  EVMCALL_VALUE="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000068e71aa2000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000001baf80"
  EVMCALL_QID="" # USER TO PROVIDE
  
  submit_and_check "EVMCALL GOOD" "$EVMCALL_QUERY_DATA" "1000000loya" "$EVMCALL_VALUE" "$EVMCALL_QID" "250000"
else
  echo "⚠️  WARNING: EVMCall data spec is not registered. Skipping EVMCALL GOOD test."
  echo "To register EVMCall, run the command in start_a_chain.sh (lines 336-343)"
  TEST_NAMES+=("EVMCALL GOOD")
  TEST_RESULTS+=("SKIPPED")
  echo ""
fi

# Create first reporter for bad eth/usd report
BAD_ETH_REPORTER_NAME="bad_eth_reporter_$(date +%s)_$$"
create_reporter_account "$BAD_ETH_REPORTER_NAME" "bad eth/usd report"

# CYCLE LIST BAD (eth/usd) - using new account
ETH_QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003657468000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
ETH_VALUE="000000000000000000000000000000000000000000000084bd26b6c2dd7c0000"
ETH_QID="83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992"

submit_and_check "CYCLE LIST BAD (eth/usd)" "$ETH_QUERY_DATA" "10000loya" "$ETH_VALUE" "$ETH_QID" "" "$BAD_ETH_REPORTER_NAME"

# IN TELLIOT, NOT CONFIG GOOD (sfrxusd/usd)
SFRXUSD_QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000007736672787573640000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
SFRXUSD_VALUE="00000000000000000000000000000000000000000000000010308da8b2f73e00"
SFRXUSD_QID="ab30caa3e7827a27c153063bce02c0b260b29c0c164040c003f0f9ec66002510"

submit_and_check "IN TELLIOT, NOT CONFIG GOOD (sfrxusd/usd)" "$SFRXUSD_QUERY_DATA" "10000loya" "$SFRXUSD_VALUE" "$SFRXUSD_QID"

# Create second reporter for bad ltc/usd report
BAD_LTC_REPORTER_NAME="bad_ltc_reporter_$(date +%s)_$$"
create_reporter_account "$BAD_LTC_REPORTER_NAME" "bad ltc/usd report"

# IN TELLIOT, NOT CONFIG BAD (ltc/usd) - using new account
LTC_QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c00000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000036c7463000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
LTC_VALUE="00000000000000000000000000000000000000000000000010308da8b2f73e00"
LTC_QID="19585d912afb72378e3986a7a53f1eae1fbae792cd17e1d0df063681326823ae"

submit_and_check "IN TELLIOT, NOT CONFIG BAD (ltc/usd)" "$LTC_QUERY_DATA" "10000loya" "$LTC_VALUE" "$LTC_QID" "" "$BAD_LTC_REPORTER_NAME"

# IN CONFIG, NOT TELLIOT (soup/usd) (add to LVM config to test)
SOUP_QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000004736f75700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000"
SOUP_VALUE="0000000000000000000000000000000000000000000000000de0b6b3a7640000"
SOUP_QID="dddaf631c368e4728f2a68e1feb705e4b2a2bdc7bd636c5f2b94bcd8bad48677"

submit_and_check "IN CONFIG, NOT TELLIOT (soup/usd)" "$SOUP_QUERY_DATA" "10000loya" "$SOUP_VALUE" "$SOUP_QID"

# 0x62733e63499a25E35844c91275d4c3bdb159D29d
# Legacy TRBBridge deposit 142 should fail
LEGACY_TRBRIDGE_QUERY_DATA="000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000009545242427269646765000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008e"
TRBRIDGE_VALUE="000000000000000000000000e4746dd0b7d785766405fcf909953ce4f50fb53400000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000008ac7230489e800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002d74656c6c6f72313878373475716875326b6b6872333278763870323436726636616c3565617966657a6a6b6a6b00000000000000000000000000000000000000"
submit_and_expect_failure "TRBBRIDGE V1 FAILS" "$LEGACY_TRBRIDGE_QUERY_DATA" "$TRBRIDGE_VALUE" "cannot report deprecated trbbridge queries" "250000"

if ensure_trbbridgev2_spec_registered; then
  # TRBBridgeV2 deposit 142 should succeed
  TRBRIDGEV2_QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000b545242427269646765563200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000008e"
  TRBRIDGEV2_QID="b2e283e744cc05ab396ee614e38488474cac932643e482b28a4a39246afbdfc5"
  submit_and_check "TRBBRIDGEV2 SUCCEEDS" "$TRBRIDGEV2_QUERY_DATA" "" "$TRBRIDGE_VALUE" "$TRBRIDGEV2_QID" "250000"
else
  echo "⚠️  WARNING: TRBBridgeV2 data spec is not available. Skipping bridge tests."
  TEST_NAMES+=("TRBBRIDGEV2 SUCCEEDS")
  TEST_RESULTS+=("SKIPPED")
  echo ""
fi

# NFLWINNER
echo "Checking if NFLWinner data spec is registered..."
if ./layerd query registry data-spec NFLWinner --home $HOME_DIR/alice &>/dev/null; then
  echo "NFLWinner data spec found, proceeding with NFLWINNER GOOD test..."
  
  NFLWINNER_QUERY_DATA="0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000094e464c57696e6e6572000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000009000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000005636f6c74730000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008737465656c657273000000000000000000000000000000000000000000000000"
  NFLWINNER_VALUE="00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000008737465656c657273000000000000000000000000000000000000000000000000"
  NFLWINNER_QID="50be2bfaaa192044b51c734e9b6851c201d283a17856f6c137b2e4b44a0edd20"
  
  submit_and_check "NFLWINNER (not in telliot or config)" "$NFLWINNER_QUERY_DATA" "10000loya" "$NFLWINNER_VALUE" "$NFLWINNER_QID"
else
  echo "⚠️  WARNING: NFLWinner data spec is not registered. Skipping NFLWINNER test."
  TEST_NAMES+=("NFLWINNER")
  TEST_RESULTS+=("SKIPPED")
  echo ""
fi

# EVMCALL BAD (run last, wait 6 seconds before starting)
echo "Waiting a few seconds before starting EVMCALL BAD test..."
sleep 3

if ./layerd query registry data-spec EVMCall --home $HOME_DIR/alice &>/dev/null; then
  # EVMCALL BAD - using alice
  EVMCALL_QUERY_DATA="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000745564d43616c6c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000010000000000000000000000005589e306b1920f009979a50b88cae32aecd471e4000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000047f629c6500000000000000000000000000000000000000000000000000000000"
  EVMCALL_VALUE="00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000068e71aa2000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000002bad00"
  EVMCALL_QID="" # USER TO PROVIDE
  submit_and_check "EVMCALL BAD" "$EVMCALL_QUERY_DATA" "1000000loya" "$EVMCALL_VALUE" "$EVMCALL_QID" "250000" "alice"
else
  echo "⚠️  WARNING: EVMCall data spec is not registered. Skipping EVMCALL BAD test."
  TEST_NAMES+=("EVMCALL BAD")
  TEST_RESULTS+=("SKIPPED")
  echo ""
fi

echo "=========================================="
echo "           TEST SUMMARY"
echo "=========================================="
echo ""

# Display results table
for i in "${!TEST_NAMES[@]}"; do
  name="${TEST_NAMES[$i]}"
  result="${TEST_RESULTS[$i]}"
  
  if [ "$result" == "SUCCESS" ]; then
    echo "✅ $name - SUCCESS"
  elif [ "$result" == "SKIPPED" ]; then
    echo "⚠️  $name - SKIPPED"
  else
    echo "❌ $name - FAILED"
  fi
done

echo ""
echo "=========================================="

# Count successes, failures, and skipped
success_count=0
failed_count=0
skipped_count=0
for result in "${TEST_RESULTS[@]}"; do
  if [ "$result" == "SUCCESS" ]; then
    ((success_count++))
  elif [ "$result" == "SKIPPED" ]; then
    ((skipped_count++))
  else
    ((failed_count++))
  fi
done

total_count=$((success_count + failed_count + skipped_count))
echo "Total: $total_count | Passed: $success_count | Failed: $failed_count | Skipped: $skipped_count"
echo "=========================================="