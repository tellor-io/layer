#!/bin/bash

# This script starts a Layer-app, creates a validator with the provided parameters, then
# keeps running it validating blocks.
echo "LAYERD_NODE_HOME: ${LAYERD_NODE_HOME}"
# check if environment variables are set
if [[ -z "${LAYERD_NODE_HOME}" || -z "${MONIKER}" ]]
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
    layerd tx staking create-validator /${LAYERD_NODE_HOME}/config/${MONIKER}.json \
    --chain-id="layer" \
    --from="${MONIKER}" \
    --keyring-backend="test" \
    --home="${LAYERD_NODE_HOME}" \
    --fees="5000loya" \
    --keyring-dir="${LAYERD_NODE_HOME}" \
    --yes
    output=$(layerd query staking validator "${VAL_ADDRESS}" 2>/dev/null)
    if [[ -n "${output}" ]] ; then
      break
    fi
    echo "trying to create validator..."
    sleep 1s
  done
} &

# start node
layerd start \
--home="${LAYERD_NODE_HOME}" \
--moniker="${MONIKER}" \
--p2p.persistent_peers=46caafaef9237b2015dca76e5b3e3ae5109736fe@core0:26656 \
--rpc.laddr=tcp://0.0.0.0:26657