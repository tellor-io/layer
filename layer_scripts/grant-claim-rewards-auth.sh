#!/bin/bash

# clear the terminal
clear

# Stop execution if any command fails
set -e

### YOU MUST BE INSIDE OF THE LAYER FOLDER TO RUN THIS SCRIPT (~/layer)
## RUN WITH sh ./layer_scripts/grant-claim-rewards-auth.sh

## YOU WILL NEED TO SET THIS TO WHATEVER NODE YOU WOULD LIKE TO USE
export LAYER_NODE_URL=https://node-palmito.tellorlayer.com/rpc/
export KEYRING_BACKEND="file"
export GRANTER_ADDR_ONE="tellor1zhg69su8p5zplr7jkzkav7874ekncn83592rlk"
export GRANTER_ADDR_TWO="tellor1hu8vk2zzyety0j4c88vpyfew8rc7frqnjf63nm"
export GRANTER_ADDR_THREE="tellor1p7y0yvqnw3ajjq6vsjv7yahsvjxs6ahhs5lxq7"
export GRANTER_ADDR_FOUR="tellor10zjsg0205re8g74t6qqqd0jh4nm8cuyy4nw99g"
export GRANTER_ADDR_FIVE="tellor1uasku64eztzzwne8gx58kzxkfn5lu803ekfqpn"
export GRANTER_ADDR_SIX="tellor1edr39pfjd2j0l7335dx2zkgk5zmtl5cwnuh6xw"

# just grant access to one account and we can have a script running that calls it for all of them
export GRANTEE_ADDR="tellor10usyr7v4xe2uhtnvg4kwtgtuzh5e4u2378zjj9"

# If you already have layer built comment out the git commands and the go build command
git checkout main
git pull
git checkout tags/v4.0.3
go build ./cmd/layerd


# Grant access from all accounts to the grantee account for the MsgWithdrawTip message.
echo "Granting access from $GRANTER_ADDR_ONE to $GRANTEE_ADDR"
./layerd tx authz grant $GRANTEE_ADDR generic  --msg-type=/layer.reporter.MsgWithdrawTip --from $GRANTER_ADDR_ONE --chain-id layertest-4 --fees 15loya --node="$LAYER_NODE_URL" --keyring-backend $KEYRING_BACKEND --yes

echo "Granting access from $GRANTER_ADDR_TWO to $GRANTEE_ADDR"
./layerd tx authz grant $GRANTEE_ADDR generic  --msg-type=/layer.reporter.MsgWithdrawTip --from $GRANTER_ADDR_TWO --chain-id layertest-4 --fees 15loya --node="$LAYER_NODE_URL" --keyring-backend $KEYRING_BACKEND --yes

echo "Granting access from $GRANTER_ADDR_THREE to $GRANTEE_ADDR"
./layerd tx authz grant $GRANTEE_ADDR generic  --msg-type=/layer.reporter.MsgWithdrawTip --from $GRANTER_ADDR_THREE --chain-id layertest-4 --fees 15loya --node="$LAYER_NODE_URL" --keyring-backend $KEYRING_BACKEND --yes

echo "Granting access from $GRANTER_ADDR_FOUR to $GRANTEE_ADDR"
./layerd tx authz grant $GRANTEE_ADDR generic  --msg-type=/layer.reporter.MsgWithdrawTip --from $GRANTER_ADDR_FOUR --chain-id layertest-4 --fees 15loya --node="$LAYER_NODE_URL" --keyring-backend $KEYRING_BACKEND --yes

echo "Granting access from $GRANTER_ADDR_FIVE to $GRANTEE_ADDR"
./layerd tx authz grant $GRANTEE_ADDR generic  --msg-type=/layer.reporter.MsgWithdrawTip --from $GRANTER_ADDR_FIVE --chain-id layertest-4 --fees 15loya --node="$LAYER_NODE_URL" --keyring-backend $KEYRING_BACKEND --yes

echo "Granting access from $GRANTER_ADDR_SIX to $GRANTEE_ADDR"
./layerd tx authz grant $GRANTEE_ADDR generic  --msg-type=/layer.reporter.MsgWithdrawTip --from $GRANTER_ADDR_SIX --chain-id layertest-4 --fees 15loya --node="$LAYER_NODE_URL" --keyring-backend $KEYRING_BACKEND --yes

echo "All access grants have been granted"

# Claim rewards from all accounts


