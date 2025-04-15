#!/bin/bash

# Define validator addresses to remove


# Create a temporary file for the totals
# calculate how many tokens need to be removed and where they should be removed from to be used in the next piece of the script
# find address for bonded tokens pool and distribution module
# calculate total tokens removed from supply
jq -r '
  (reduce (
    .app_state.distribution.outstanding_rewards[] |
    select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv") |
    .outstanding_rewards[0].amount // "0"
  ) as $amount (0; . + ($amount | tonumber))) as $outstanding_validator_rewards_distribution |

  (reduce (
    .app_state.staking.delegations[] |
    select(.validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv") |
    .shares // "0"
  ) as $amount (0; . + ($amount | tonumber))) as $bonded_delegations_deleted_amount |

  (reduce (
    .app_state.staking.delegations[] |
    select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9") |
    .shares // "0"
  ) as $amount (0; . + ($amount | tonumber))) as $non_bonded_delegations_deleted_amount |

  (reduce (
    .app_state.staking.last_validator_powers[] |
    select(.address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv") |
    .power // "0"
  ) as $amount (0; . + ($amount | tonumber))) as $last_validators_power_removed |

  (.app_state.auth.accounts[] | select(.name == "distribution") | .base_account.address) as $distribution_module_address |

  (.app_state.auth.accounts[] | select(.name == "bonded_tokens_pool") | .base_account.address) as $bonded_tokens_pool_address |

  (.app_state.auth.accounts[] | select(.name == "not_bonded_tokens_pool") | .base_account.address) as $not_bonded_tokens_pool_address |

  .app_state.bank.supply[0].amount as $total_supply |

  .app_state.staking.last_total_power as $last_total_power |

  (.app_state.bank.balances[] | select(.address == $distribution_module_address) | .coins[0].amount) as $distribution_module_balance |

  (.app_state.bank.balances[] | select(.address == $bonded_tokens_pool_address) | .coins[0].amount) as $bonded_tokens_pool_balance |

  (.app_state.bank.balances[] | select(.address == $not_bonded_tokens_pool_address) | .coins[0].amount) as $not_bonded_tokens_pool_balance |

  {
    "outstanding_validator_rewards_distribution": $outstanding_validator_rewards_distribution,
    "bonded_delegations_deleted_amount": $bonded_delegations_deleted_amount,
    "non_bonded_delegations_deleted_amount": $non_bonded_delegations_deleted_amount,
    "last_validators_power_removed": $last_validators_power_removed,
    "distribution_module_address": $distribution_module_address,
    "bonded_tokens_pool_address": $bonded_tokens_pool_address,
    "not_bonded_tokens_pool_address": $not_bonded_tokens_pool_address,
    "total_supply": $total_supply,
    "last_total_power": $last_total_power,
    "distribution_module_balance": $distribution_module_balance,
    "bonded_tokens_pool_balance": $bonded_tokens_pool_balance,
    "not_bonded_tokens_pool_balance": $not_bonded_tokens_pool_balance
  } |
  
  tostring
' ./exported_state_test.json > removed_tokens.json

# Display the totals
echo "Token amounts removed:"
cat removed_tokens.json

jq '.outstanding_validator_rewards_distribution |= floor | .bonded_delegations_deleted_amount |= floor | .non_bonded_delegations_deleted_amount |= floor' removed_tokens.json > temp.json && mv temp.json removed_tokens.json

outstanding_validator_rewards_distribution=$(jq -r '.outstanding_validator_rewards_distribution' removed_tokens.json)
bonded_delegations_deleted_amount=$(jq -r '.bonded_delegations_deleted_amount' removed_tokens.json)
non_bonded_delegations_deleted_amount=$(jq -r '.non_bonded_delegations_deleted_amount' removed_tokens.json)
last_validators_power_removed=$(jq -r '.last_validators_power_removed' removed_tokens.json)
distribution_module_address=$(jq -r '.distribution_module_address' removed_tokens.json)
bonded_tokens_pool_address=$(jq -r '.bonded_tokens_pool_address' removed_tokens.json)
not_bonded_tokens_pool_address=$(jq -r '.not_bonded_tokens_pool_address' removed_tokens.json)

# Now perform the actual removal and save to the original file
# remove from consensus.validators
# remove all delegations to these validators in staking.delegations
# Update, slash, and jail validators from staking.validators
# remove validators from staking.last_validator_powers
# Update staking.last_total_power
# remove validators from slashing.missed_blocks
# Jail until 2999 and tombstone validators from slashing.signing_infos
# remove all delegations to validators in distribution.delegator_starting_infos
# Remove all outstanding validator rewards
# Update distribution account balance
# Update bonded_tokens_pool account balance
# update total supply
# remove from distribution.validator_slash_events
# update vote extensions enabled height
# update genesis time 
jq --argjson outstanding_validator_rewards_distribution $outstanding_validator_rewards_distribution \
  --argjson bonded_delegations_deleted_amount $bonded_delegations_deleted_amount \
  --argjson non_bonded_delegations_deleted_amount $non_bonded_delegations_deleted_amount \
  --argjson last_validators_power_removed $last_validators_power_removed \
  --arg distribution_module_address "$distribution_module_address" \
  --arg bonded_tokens_pool_address "$bonded_tokens_pool_address" \
  --arg not_bonded_tokens_pool_address "$not_bonded_tokens_pool_address" \
  '
  .consensus.validators -= [.consensus.validators[] | 
  select(.name == "luke-moniker" or .name == "yoda-moniker" or .name == "palpatine-moniker" or .name == "darth_maul-moniker")] |

  .app_state.staking.delegations -= [.app_state.staking.delegations[] | 
  select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |

  .app_state.staking.validators[] |= (
    if .operator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or 
      .operator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or 
      .operator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or 
      .operator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv" 
    then .
      .jailed = true |
      .status = "BOND_STATUS_UNBONDED" |
      .tokens = "0" |
      .delegator_shares = "0"
    else 
      . 
    end) |

  .app_state.staking.last_validator_powers -= [.app_state.staking.last_validator_powers[] | 
  select(.address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |

  .app_state.slashing.missed_blocks -= [.app_state.slashing.missed_blocks[] | 
  select(.address == "tellorvalcons1qsyw9f02sxlqqls490vnqvdy2sm5uw48m83q56" or .address == "tellorvalcons1cqwtrppf6frfwkt0lfv0070sstt463kcprln8w" or .address == "tellorvalcons1dckkr3sve3cux9wnkxrnauzr064q07eles6yk4" or .address == "tellorvalcons1cpq4g5ljhh9y035ztkvnyz9p77ypahxxktpglm")] |

  .app_state.slashing.signing_infos[] |= (
  if .address == "tellorvalcons1qsyw9f02sxlqqls490vnqvdy2sm5uw48m83q56" or 
    .address == "tellorvalcons1cqwtrppf6frfwkt0lfv0070sstt463kcprln8w" or 
    .address == "tellorvalcons1dckkr3sve3cux9wnkxrnauzr064q07eles6yk4" or 
    .address == "tellorvalcons1cpq4g5ljhh9y035ztkvnyz9p77ypahxxktpglm" 
  then 
    .validator_signing_info.jailed_until = "2999-04-09T09:27:10.698086179Z" | 
    .validator_signing_info.tombstoned = true 
  else 
    . 
  end) |

  .app_state.distribution.delegator_starting_infos -= [.app_state.distribution.delegator_starting_infos[] | 
  select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |   

  .app_state.distribution.outstanding_rewards -= [.app_state.distribution.outstanding_rewards[] | 
  select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |

  .app_state.distribution.validator_accumulated_commissions -= [.app_state.distribution.validator_accumulated_commissions[] | 
  select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |   

  .app_state.distribution.validator_historical_rewards -= [.app_state.distribution.validator_historical_rewards[] | 
  select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |

  .app_state.distribution.validator_current_rewards -= [.app_state.distribution.validator_current_rewards[] | 
  select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |

  .app_state.distribution.validator_slash_events -= [.app_state.distribution.validator_slash_events[] | 
  select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")] |

  .app_state.staking.last_total_power = ((.app_state.staking.last_total_power | tonumber) - ($last_validators_power_removed | tonumber) | tostring) |

  .app_state.bank.balances[] |= (if .address == $distribution_module_address then .coins[0].amount = ((.coins[0].amount | tonumber) - ($outstanding_validator_rewards_distribution | tonumber) | tostring) else . end) |

  .app_state.bank.balances[] |= (if .address == $bonded_tokens_pool_address then .coins[0].amount = ((.coins[0].amount | tonumber) - ($bonded_delegations_deleted_amount | tonumber) | tostring) else . end) |

  .app_state.bank.balances[] |= (if .address == $not_bonded_tokens_pool_address then .coins[0].amount = ((.coins[0].amount | tonumber) - ($non_bonded_delegations_deleted_amount | tonumber) | tostring) else . end) |

  .app_state.bank.supply[0].amount = ((.app_state.bank.supply[0].amount | tonumber) - ($outstanding_validator_rewards_distribution | tonumber) - ($bonded_delegations_deleted_amount | tonumber) - ($non_bonded_delegations_deleted_amount | tonumber) | tostring) |
  
  .consensus.params.abci.vote_extensions_enabled_height = ((.initial_height | tonumber) + 1 | tostring)
' ./exported_state_test.json > test_state_edited.json

echo "State file updated and validators have been removed. Make sure to update the genesis time with an appropriate time for when you can start the chain back up"
echo "Doing some queries on the new json document to get the data that should have updated"

echo "Last Totel power after edit: "
jq '.app_state.staking.last_total_power' test_state_edited.json

echo "staking.Validators after edit: "
jq '.app_state.staking.validators[] | select(.operator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .operator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .operator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .operator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")' test_state_edited.json

echo "slashing.signing_infos after edit: "
jq '.app_state.slashing.signing_infos[] | select(.address == "tellorvalcons1qsyw9f02sxlqqls490vnqvdy2sm5uw48m83q56" or .address == "tellorvalcons1cqwtrppf6frfwkt0lfv0070sstt463kcprln8w" or .address == "tellorvalcons1dckkr3sve3cux9wnkxrnauzr064q07eles6yk4" or .address == "tellorvalcons1cpq4g5ljhh9y035ztkvnyz9p77ypahxxktpglm")' test_state_edited.json

echo "bonded tokens pool balance after edit: "
jq --arg bonded_tokens_pool_address "$bonded_tokens_pool_address" '.app_state.bank.balances[] | select(.address == $bonded_tokens_pool_address)' test_state_edited.json

echo "distribution module balance after edit: "
jq --arg distribution_module_address "$distribution_module_address" '.app_state.bank.balances[] | select(.address == $distribution_module_address)' test_state_edited.json

echo "not bonded tokens pool balance after edit: "
jq --arg not_bonded_tokens_pool_address "$not_bonded_tokens_pool_address" '.app_state.bank.balances[] | select(.address == $not_bonded_tokens_pool_address)' test_state_edited.json

echo "total supply after edit: "
jq '.app_state.bank.supply[0].amount' test_state_edited.json

echo "distribution.delegator_starting_infos after edit: "
jq '.app_state.distribution.delegator_starting_infos[] | select(.validator_address == "tellorvaloper1hwez285ve3z52mx95dnh5ad0yup453vfy6wzkz" or .validator_address == "tellorvaloper1q8m9lqc7ajmgg3kl60n0jrqejvv4yasl4th5q9" or .validator_address == "tellorvaloper1zldjml93erklsehcv4nsuh6a0rf02nvyefqn7h" or .validator_address == "tellorvaloper18e3ct9zrakk2w4zvz0qg3228s6dwr0kqwjgkgv")' test_state_edited.json

