#!/bin/bash

# This script creates the necessary files before starting layerd

# only create the priv_validator_state.json if it doesn't exist and the command is start
if [[ $1 == "start" && ! -f ${LAYER_HOME}/data/priv_validator_state.json ]]
then
    mkdir -p ${LAYER_HOME}/data
    cat <<EOF > ${LAYER_HOME}/data/priv_validator_state.json
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF
fi

echo "Starting layerd with command:"
echo "/bin/layerd $@"
echo ""

exec /bin/layerd $@