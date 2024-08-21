#!/bin/bash

# This script starts a Layer-app, creates a validator with the provided parameters, then
# keeps running it validating blocks.
echo "LAYERD_NODE_HOME: ${LAYERD_NODE_HOME}"
# check if environment variables are set
if [[ -z "${LAYERD_NODE_HOME}" || -z "${MONIKER}" || -z "${AMOUNT}" || -z "${COMMISSION_RATE}" || -z "${MIN_TOKENS_REQUIRED}" ]]
then
  echo "Environment not setup correctly. Please set: LAYERD_NODE_HOME, MONIKER, AMOUNT variables"
  exit 1
fi

# create necessary structure if doesn't exist
if [[ ! -f ${LAYERD_NODE_HOME}/data/priv_validator_state.json ]]
then
    mkdir "${LAYERD_NODE_HOME}"/data
    cat <<EOF > ${LAYERD_NODE_HOME}/data/priv_validator_state.json
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF
fi

# write create validator json if it doesn't exist
if [[ ! -f ${LAYERD_NODE_HOME}/${MONIKER}.json ]]
then
    pubkey=$(layerd comet show-validator --home  ${LAYERD_NODE_HOME})
    cat <<EOF > ${LAYERD_NODE_HOME}/${MONIKER}.json
{
    "pubkey": $pubkey,
    "amount": "$AMOUNT",
    "moniker": "$MONIKER",
    "commission-rate": "0.10",
    "commission-max-rate": "0.20",
    "commission-max-change-rate": "0.01",
    "min-self-delegation": "1"
}
EOF
fi

{
  # wait for the node to get up and running
  while true
  do
    status_code=$(curl --write-out '%{http_code}' --silent --output /dev/null localhost:26657/status)
    if [[ "${status_code}" -eq 200 ]] ; then
      break
    fi
    echo "Waiting for node to be up..."
    sleep 2s
  done

  VAL_ADDRESS=$(layerd keys show "${MONIKER}" --keyring-backend test --bech=val --home  ${LAYERD_NODE_HOME} --keyring-dir ${LAYERD_NODE_HOME} -a)
  # keep retrying to create a validator
  while true
  do
    # create validator
    layerd tx staking create-validator /${LAYERD_NODE_HOME}/${MONIKER}.json \
    --chain-id="layer" \
    --from="${MONIKER}" \
    --keyring-backend="test" \
    --home="${LAYERD_NODE_HOME}" \
    --keyring-dir="${LAYERD_NODE_HOME}" \
    --gas-prices="1loya" \
    --yes
    output=$(layerd query staking validator "${VAL_ADDRESS}" 2>/dev/null)
    if [[ -n "${output}" ]] ; then
      break
    fi
    echo "trying to create validator..."
    sleep 1s
  done

  REPORTER=$(layerd keys show "${MONIKER}" --keyring-backend test --home  ${LAYERD_NODE_HOME} --keyring-dir ${LAYERD_NODE_HOME} -a)
  while true
  do
  layerd tx reporter create-reporter "${COMMISSION_RATE}" "${MIN_TOKENS_REQUIRED}"  \
  --from=${MONIKER} \
  --keyring-backend="test" \
  --keyring-dir="${LAYERD_NODE_HOME}" \
  --chain-id="layer" \
  --gas-prices="1loya" \
  --yes

  selector=$(layerd query reporter selector-reporter "${REPORTER}" 2>/dev/null)
    if [[ -n "${selector}" ]] ; then
      break
    fi
    echo "trying to create reporter..."
    sleep 1s
  done
} &

# start node
layerd start \
--home="${LAYERD_NODE_HOME}" \
--moniker="${MONIKER}" \
--key-name="${MONIKER}" \
--keyring-backend="test" \
--p2p.persistent_peers=46caafaef9237b2015dca76e5b3e3ae5109736fe@core0:26656 \
--rpc.laddr=tcp://0.0.0.0:26657 \
--api.enable \
--api.swagger \
--panic-on-daemon-failure-enabled=false