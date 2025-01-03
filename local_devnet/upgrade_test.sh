#!/bin/bash

# THIS SCRIPT CAN BE USED TO TEST A CHAIN UPGRADE WHILE RUNNING THE LOCAL DEVNET
# BE SURE TO SET "expedited": false FOR THE PROPOSAL. 
# THE LOCAL DEVNET HAS A 15s VOTING PERIOD. 
# RUN THIS SCRIPT AS SOON AS THE DOCKER CONTAINERS START UP.
# Sleep times may need to be adjusted depending on the power of your computer. (faster computers go faster)

# Stop execution if any command fails
set -e

# use docker ps to get the container id hashes
echo "setting variables"
export ETH_RPC_URL="https://sepolia.infura.io/v3/your_key_here"
export TOKEN_BRIDGE_CONTRACT="0xFC1C57F1E466605e3Dd40840bC3e7DdAa400528c"
export upgrade_binary_path="/path/to/layerd"

# Get all container IDs into an array
container_ids=($(docker ps -q))

# automatically sets a CONTAINER_ variable for all containers
# (for running docker exec)
for i in "${!container_ids[@]}"; do
  varname="CONTAINER_$((i + 1))"
  export "$varname=${container_ids[i]}"
  echo "Exported $varname=${container_ids[i]}"
done

# optionally view the logs in the terminal:
# mac os:
# osascript -e "tell application \"Terminal\" to do script \"docker logs -f $CONTAINER_1\""
# osascript -e "tell application \"Terminal\" to do script \"docker logs -f $CONTAINER_3\""
# desktop linux with gnome:
# gnome-terminal -- bash -c "docker logs -f $CONTAINER_1; exec bash"
# gnome-terminal -- bash -c "docker logs -f $CONTAINER_NAME; exec bash"

# copy proposal to node1 container and submit proposal
echo "copying the proposal to CONTAINER_1"
docker cp ./proposal.json $CONTAINER_1:/bin/

echo "proposing the upgrade"
docker exec $CONTAINER_1 layerd tx gov submit-proposal /bin/proposal.json --from validator --chain-id layer-1 --home /var/cosmos-chain/layer-1 --keyring-backend test --fees 510loya --yes

# (optionally) check if proposal is live
# docker exec $node1_id layerd query gov proposals
# wait a bit
echo "voting in the next block..."
sleep 5

# vote on proposal
echo "voting on the upgrade proposal"
docker exec $CONTAINER_1 layerd tx gov vote 1 yes --from validator --chain-id layer-1 --home /var/cosmos-chain/layer-1 --keyring-backend test --fees 500loya --yes
docker exec $CONTAINER_2 layerd tx gov vote 1 yes --from validator --chain-id layer-1 --home /var/cosmos-chain/layer-1 --keyring-backend test --fees 500loya --yes
docker exec $CONTAINER_3 layerd tx gov vote 1 yes --from validator --chain-id layer-1 --home /var/cosmos-chain/layer-1 --keyring-backend test --fees 500loya --yes
docker exec $CONTAINER_4 layerd tx gov vote 1 yes --from validator --chain-id layer-1 --home /var/cosmos-chain/layer-1 --keyring-backend test --fees 500loya --yes

echo "making reporters in the next block..."
sleep 5

# create 2 reporters to sanity check that reporting works before and after the upgrade
echo "creating two reporters to test that reporting works before / after upgrade"
docker exec $CONTAINER_3 layerd tx reporter create-reporter "2000000" "10000000" --from validator --home /var/cosmos-chain/layer-1 --keyring-dir /var/cosmos-chain/layer-1 --keyring-backend test --chain-id layer-1 --fees 500loya --yes
docker exec $CONTAINER_4 layerd tx reporter create-reporter "200000" "1000000" --from validator --home /var/cosmos-chain/layer-1 --keyring-dir /var/cosmos-chain/layer-1 --keyring-backend test --chain-id layer-1 --fees 500loya --yes

# wait for chain to stop
sleep 30

#copy new binary into each container
docker cp $upgrade_binary_path $CONTAINER_1:/bin/
docker cp $upgrade_binary_path $CONTAINER_2:/bin/
docker cp $upgrade_binary_path $CONTAINER_3:/bin/
docker cp $upgrade_binary_path $CONTAINER_4:/bin/

# all done! 
echo "Done!"
echo "Restart the docker containers via docker desktop to verify that the upgrade was successsful."
