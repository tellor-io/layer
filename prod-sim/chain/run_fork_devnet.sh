#!/bin/bash

# Get current timestamp in the required format
GENESIS_TIME=$(date -u +"%Y-%m-%dT%H:%M:%S.000000000Z")

# Get timestamp for tomorrow in the required format
FUTURE_TIME=$(date -u -v+1d +"%Y-%m-%dT%H:%M:%S.000000000Z")

# Set the path to layerd binary
LAYERD_PATH="/Users/caleb/layer/layerd"

# Create necessary directories
mkdir -p bin genesis

PUBLIC_GENESIS="./exported_data/public_exported_genesis.json"
LOCAL_GENESIS="./exported_data/local_docker_genesis.json"
DOCKER_DATA="./exported_data/docker_module_data.json"
REPORTER_DATA="./exported_data/reporter_module_state.json"
OUTPUT_GENESIS="./genesis/genesis.json"

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo "Error: You have uncommitted changes in your working directory."
    echo "Please commit or stash your changes before running this script."
    exit 1
fi

echo "ensure no old docker containers are running..."
docker compose down -v

# Store the current branch name
CURRENT_BRANCH=$(git branch --show-current)

echo "checking out temp-fork-devnet-update branch"
# Create and checkout a temporary branch
git checkout -b temp-fork-devnet-update

echo "replace ./app/upgrades.go with template so fork upgrade is set up"
cp ./fork-file-templates/app_upgrades_go_template.txt ../../app/upgrades.go

echo "edit module.go for oracle module to update consensus version and register upgrade handler"
python3 ./update_consensus.py ../../x/oracle/module.go
echo "replace oracle migrations.go with template..."
cp ./fork-file-templates/oracle_fork_migrations_template.txt ../../x/oracle/keeper/migrations.go

echo "edit module.go for reporter module to update consensus version"
python3 ./update_consensus.py ../../x/reporter/module/module.go
echo "replace reporter migrations.go with template..."
cp ./fork-file-templates/reporter_migrations_go_template.txt ../../x/reporter/keeper/migrations.go

echo "edit module.go for dispute module to update consensus version"
python3 ./update_consensus.py ../../x/dispute/module.go
echo "replace dispute migrations.go with template..."
cp ./fork-file-templates/dispute_fork_migrations_template.txt ../../x/dispute/keeper/migrations.go

# switch from chain directory to repo root to build layerd binary
cd ../..

echo "add edited files to git..."
git add ./app/upgrades.go
git add ./x/oracle/module.go
git add ./x/oracle/keeper/migrations.go
git add ./x/reporter/module/module.go
git add ./x/reporter/keeper/migrations.go
git add ./x/dispute/module.go
git add ./x/dispute/keeper/migrations.go
git commit -m "update consensus version for fork upgrade"

echo "build chain binary from temp-fork-devnet-update branch for docker environment..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./prod-sim/chain/bin/layerd ./cmd/layerd

echo "checkout back to $CURRENT_BRANCH branch"
git checkout $CURRENT_BRANCH

# Clean up the temporary branch and its changes
echo "cleaning up temporary branch..."
git branch -D temp-fork-devnet-update
git reset --hard HEAD
git clean -fd

sleep 3

# switch back to chain directory
cd prod-sim/chain

echo "edit and merge genesis file to replace validator set"
jq -s --arg genesis_time "$GENESIS_TIME" --arg future_time "$FUTURE_TIME" '.[0] as $public | .[1] as $local | .[2] as $docker_data |
$public |
.consensus.params.abci.vote_extensions_enable_height = ((.initial_height + 1) | tostring) |
($local.app_state.staking.validators | map(.operator_address)) as $local_val_ops |
($local.app_state.bank.balances | map(select(.address | IN($local.app_state.auth.accounts | map(select(.["@type"] == "/cosmos.auth.v1beta1.BaseAccount")) | .[].address)))) as $new_balances |
($new_balances | map(.coins[].amount | tonumber) | add) as $new_amount_sum |
.consensus.validators = $local.consensus.validators |
.app_state.staking.last_validator_powers = $local.app_state.staking.last_validator_powers |
.app_state.staking.last_total_power = $local.app_state.staking.last_total_power |
.app_state.auth.accounts = (.app_state.auth.accounts + ($local.app_state.auth.accounts | map(select(.["@type"] == "/cosmos.auth.v1beta1.BaseAccount")))) |

(reduce .app_state.staking.validators[] as $validator (
  {};
  . + {
    ($validator.operator_address): (
      if ($validator.delegator_shares | tonumber) > 0 then
        ($validator.tokens | tonumber) / ($validator.delegator_shares | tonumber)
      else 
        0
      end
    )
  }
)) as $validator_exchange_rates |

(reduce (
  .app_state.staking.delegations[] |
  {
    shares: (.shares | tonumber),
    validator: .validator_address
  }
) as $delegation (
  0; 
  . + ($delegation.shares * (($validator_exchange_rates[$delegation.validator] // 0) | tonumber) | floor)
)) as $total_staked_tokens |

.app_state.bank.balances = (
 .app_state.bank.balances | map(
 if .address == "tellor1tygms3xhhs3yv487phx3dw4a95jn7t7lpdv94k" then
  .coins[0].amount = ($total_staked_tokens | tostring)
 elif .address == "tellor1fl48vsnmsdzcv85q5d2q4z5ajdha8yu34ds5rz" then
  .coins[0].amount = "3000000"
 else
  .
 end
 )
) |
.app_state.bank.balances = (.app_state.bank.balances + $new_balances) |
.app_state.bank.supply[].amount = ((.app_state.bank.balances | map(.coins[].amount | tonumber) | add) | tostring) |
.app_state.staking.validators = [
  (.app_state.staking.validators[] | 
   . + {
     "jailed": true,
     "status": "BOND_STATUS_UNBONDED",
     "tokens": .tokens,
     "delegator_shares": .delegator_shares
   }),
  $local.app_state.staking.validators[]
] |
.app_state.slashing.signing_infos = [
  (.app_state.slashing.signing_infos[] |
   . + {
     "validator_signing_info": (.validator_signing_info + {
       "jailed_until": "2025-05-31T23:59:59Z"
     })
   }),
  $local.app_state.slashing.signing_infos[]
] |
.app_state.staking.delegations = (.app_state.staking.delegations + $local.app_state.staking.delegations) |
.app_state.slashing.missed_blocks = (.app_state.slashing.missed_blocks + $local.app_state.slashing.missed_blocks) |
.app_state.reporter.selectorTips = (.app_state.reporter.selectorTips + $local.app_state.reporter.selectorTips) |
.app_state.distribution.delegator_starting_infos = (.app_state.distribution.delegator_starting_infos + $local.app_state.distribution.delegator_starting_infos) |
.app_state.distribution.outstanding_rewards = (.app_state.distribution.outstanding_rewards + ($local.app_state.distribution.outstanding_rewards | map(.outstanding_rewards[].amount = "0"))) |
.app_state.distribution.validator_accumulated_commissions = (.app_state.distribution.validator_accumulated_commissions + ($local.app_state.distribution.validator_accumulated_commissions | map(.accumulated.commission[].amount = "0"))) |
.app_state.distribution.validator_current_rewards = (.app_state.distribution.validator_current_rewards + ($local.app_state.distribution.validator_current_rewards | map(.rewards.rewards[].amount = "0"))) |
.app_state.distribution.validator_historical_rewards = (.app_state.distribution.validator_historical_rewards + $local.app_state.distribution.validator_historical_rewards) |

.app_state.bridge.bridge_val_set = $local.app_state.bridge.bridge_val_set |
.app_state.bridge.validator_checkpoint = $local.app_state.bridge.validator_checkpoint |
.app_state.bridge.withdrawal_id = .app_state.bridge.withdrawal_id |
.app_state.bridge.operator_to_evm_address_map = (.app_state.bridge.operator_to_evm_address_map + $local.app_state.bridge.operator_to_evm_address_map) |
.app_state.bridge.evm_registered_map = (.app_state.bridge.evm_registered_map + $local.app_state.bridge.evm_registered_map) |
.app_state.bridge.bridge_valset_sigs_map = (.app_state.bridge.bridge_valset_sigs_map + $local.app_state.bridge.bridge_valset_sigs_map) |
.app_state.bridge.validator_checkpoint_params_map = (.app_state.bridge.validator_checkpoint_params_map + $local.app_state.bridge.validator_checkpoint_params_map) |
.app_state.bridge.validator_checkpoint_idx_map = (
  .app_state.bridge.validator_checkpoint_idx_map + 
  ($local.app_state.bridge.validator_checkpoint_idx_map | 
   map(.index = (($public.app_state.bridge.latest_validator_checkpoint_idx | tonumber) + 1 | tostring)))
) |
.app_state.bridge.latest_validator_checkpoint_idx = (($public.app_state.bridge.latest_validator_checkpoint_idx | tonumber) + 1 | tostring) |
.app_state.bridge.bridge_valset_by_timestamp_map = (.app_state.bridge.bridge_valset_by_timestamp_map + $local.app_state.bridge.bridge_valset_by_timestamp_map) |
.app_state.bridge.valset_timestamp_to_idx_map = (
  (.app_state.bridge.valset_timestamp_to_idx_map + 
   $local.app_state.bridge.valset_timestamp_to_idx_map) | 
  map(.index = (($public.app_state.bridge.latest_validator_checkpoint_idx | tonumber) + 1 | tostring))
) |
.app_state.bridge.deposit_id_claimed_map = (.app_state.bridge.deposit_id_claimed_map + $local.app_state.bridge.deposit_id_claimed_map) |
.app_state.gov.params.expedited_voting_period = "60s" |
.app_state.upgrade = {
  "plans": [
    {
      "name": "fork",
      "height": (.initial_height | tonumber),
      "info": "migrate over rest of module data from forked chain"
    }
  ]
} |
.genesis_time = $genesis_time |
.app_state.dispute.disputes = (.app_state.dispute.disputes | map(.dispute.dispute_end_time = $future_time)) |
.app_state.dispute.votes = (.app_state.dispute.votes | map(.vote.voteEnd = $future_time)) |
.app_state.ibc = $docker_data.ibc |
.chain_id = "tellor-devnet"' "$PUBLIC_GENESIS" "$LOCAL_GENESIS" "$DOCKER_DATA" > "$OUTPUT_GENESIS"

echo "appending reporter module data"
jq -s '.[0] as $docker_data | .[1] as $reporter_data |
{
  "reporters": ($reporter_data.reporters + $docker_data.reporters),
  "selectors": ($reporter_data.selectors + $docker_data.selectors),
  "selector_tips": ($reporter_data.selector_tips + ($docker_data.selector_tips | map(.tips = "0"))),
  "disputed_delegation_amounts": $reporter_data.disputed_delegation_amounts,
  "fee_paid_from_stake": $reporter_data.fee_paid_from_stake,
  "checksum": $reporter_data.checksum
}' "$DOCKER_DATA" "$REPORTER_DATA" >> temp.json && mv temp.json "$REPORTER_DATA"

# Function to check for error in logs and extract numbers
check_for_error() {
    ERROR_MSG=$(docker logs validator-node-0 2>&1 | grep "panic: not bonded pool balance is different from not bonded coins")
    if [ ! -z "$ERROR_MSG" ]; then
        # Extract both numbers and remove 'loya'
        FIRST_NUM=$(echo "$ERROR_MSG" | grep -o '[0-9]*loya' | head -n1 | sed 's/loya//')
        SECOND_NUM=$(echo "$ERROR_MSG" | grep -o '[0-9]*loya' | tail -n1 | sed 's/loya//')
        echo "$FIRST_NUM $SECOND_NUM"
        return 0
    fi
    return 1
}

# Function to update genesis file with extracted balance and adjust supply
update_genesis_balance() {
    local first_num=$1
    local second_num=$2
    local diff=$((second_num - first_num))
    echo "Updating genesis file with corrected balance: $second_num (difference: $diff)"
    
    # Update both the balance and supply
    jq --arg balance "$second_num" --arg diff "$diff" '
    .app_state.bank.balances |= map(
        if .address == "tellor1tygms3xhhs3yv487phx3dw4a95jn7t7lpdv94k" then
            .coins[0].amount = $balance
        else
            .
        end
    ) |
    .app_state.bank.supply[0].amount = ((.app_state.bank.supply[0].amount | tonumber) + ($diff | tonumber) | tostring)
    ' "$OUTPUT_GENESIS" > temp_genesis.json && mv temp_genesis.json "$OUTPUT_GENESIS"
}

echo "set IS_FORK=true in docker-compose.yml"
sed -i '' 's/IS_FORK=false/IS_FORK=true/g' docker-compose.yml

# Start docker compose
echo "Starting docker compose environment..."
docker compose up -d

# Wait for containers to start
sleep 3

# Monitor logs for error
echo "Monitoring validator-node-0 logs for errors..."
for i in {1..15}; do
    NUMBERS=$(check_for_error)
    if [ $? -eq 0 ]; then
        echo "Error detected in validator-node-0 logs"
        echo "Stopping docker compose environment..."
        docker compose down -v
        
        # Split the numbers and update genesis
        read -r FIRST_NUM SECOND_NUM <<< "$NUMBERS"
        update_genesis_balance "$FIRST_NUM" "$SECOND_NUM"
        
        echo "Restarting docker compose environment with updated genesis..."
        docker compose up -d
        break
    fi
    sleep 2
done

echo "Process completed. Chain is starting please wait a second before we check the logs to ensure it is running"
sleep 15
echo "Pulling the last 50 lines of validator-node-0 logs to check that the chain is running..."
docker logs validator-node-0 | grep "INF" | tail -n 50

echo "Chain is running!!! You can now interact with the chain using the layerd binary like usual."
echo "To stop the chain, run 'docker compose down -v'"
echo "To complete transactions on chain the key names are validator-{0,1,2} and make sure you use the --keyring-backend test and --keyring-dir ./prod-sim/chain/validator-info/validator-{val number} so it can find your keys"


