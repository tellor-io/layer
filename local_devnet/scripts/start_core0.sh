#!/bin/bash

# This script starts core0
echo "Starting core0..."
if [[ ! -f /root/.layer/core0/data/priv_validator_state.json ]]
then
    mkdir /root/.layer/core0/core0/data
    cat <<EOF > /root/.layer/core0/data/priv_validator_state.json
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF
fi

/bin/layerd start \
  --moniker core0 \
  --rpc.laddr tcp://0.0.0.0:26657 \
  --home /root/.layer/core0