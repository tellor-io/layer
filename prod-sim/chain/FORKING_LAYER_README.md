``Start a forked chain``

This guide will walk you through how to export the state of a running node and then use that state to start a forked chain with an entirely new validator set.

Prerequisites:
    - A node synced up with the chain (please refer to our docs on how to do that: https://docs.tellor.io/layer-docs/running-tellor-layer/node-setup)
    - Docker installed and running locally
    - Packages:
        - git
        - python3
        - go (v1.22.12)
        - jq
        - docker-compose

Steps:
    - navigate to the layer directory on your node machine
    - Stop your node that is synced up to the chain
    - Call ``./layerd export --home {ENTER_NODE_HOME_PATH} >> {enter file name}.json``
        - if you get permission errors just use sudo
    - Use ``ls`` to inspect the files. Look for dispute_module_state.json, oracle_module_state.json, reporter_module_state.json, and the exported state file.
    - Copy those files (download them from a server if need be) to the ~/layer/prod-sim/chain/exported_data folder making sure you keep the module state files the same name and then name the exported state file ``public_exported_genesis.json``
    - Set the LAYERD_PATH variable in ``~/layer/prod-sim/chain/run_fork_devnet.sh``
    - call ``git stash`` or commit your changes so that you have a clean git history
        - this script will create a temporary branch and make changes to the consensus versions/register the migration functions for the fork-upgrade. So we will be switching in between branches and need a clean git history
    - navigate to ``~/layer/prod-sim/chain`` and run ``bash ./run_fork_devnet.sh``
        - This is where all the magic happens. The code will be set up for the fork and built for the docker container. Then the exported state file will be edited to completely replace the validator set to the 3 test validators before starting up all 3 containers and starting the forked chain

- Once you have a chain running here some commands that may be useful:

    - To check the logs of the chain:
        - (docker logs -f validator-node-0) ## Can change the validator-node-0 to the number of the validator you want to check the logs of
    - To check the address of a validator account:
        - For normal account address: ./layerd keys show validator-0 -a --keyring-backend test --keyring-dir ./prod-sim/chain/validator-info/validator-0
        - For validator operator address: ./layerd keys show validator-0 --bech val --keyring-backend test --keyring-dir ./prod-sim/chain/validator-info/validator-0
    - To run tx's:
        - You can run commands like usual using the layerd binary just make sure you use the following flags:
            - --keyring-backend test
            - --keyring-dir ./prod-sim/chain/validator-info/{validator-name}
            - --chain-id tellor-devnet
            - example: ./layerd tx reporter create-reporter 0.01 1000000 test-reporter --from validator-0 --keyring-backend test --keyring-dir ./prod-sim/chain/validator-info/validator-0 --chain-id tellor-devnet --fees 10loya --yes
                - this tx will go through but should error out if you use an existing validator account as the reporter already exists
    - To query the chain:
        - run query commands like usual using the layerd binary as the docker environment is listening on port 26657
        - example: ./layerd query oracle get-current-aggregate-report 83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992




