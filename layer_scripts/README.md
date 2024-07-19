Requirements:
    - must have sed, jq, and go(v1.22) installed properly on your computer

Creating a new node:

If you are running everything in one terminal instance I recommend using screen sessions so that you can still query the chain and run other things after your node has started

Once the chain is running please select an rpc url of a current node to use in your set up of your node and set the LAYER_NODE_URL variable to that url (ex: tellor-example-node.com) WITH NO QUOTES

After finding a node url to use call "curl tellor-example-node.com:26657/status" this should return something like this:

{
    "jsonrpc":"2.0",
    "id":-1,
    "result":{
        "node_info":{
            "protocol_version":{
                "p2p":"8",
                "block":"11",
                "app":"0"
            },
            "id":"f5f6ce5d15ea80683b9133b19e245f9b27daab67",
            "listen_addr":"tcp://0.0.0.0:26656",
            "network":"layer",
            "version":"0.38.7",
            "channels":"40202122233038606100",
            "moniker":"alicemoniker",
            "other":{
                "tx_index":"on",
                "rpc_address":"tcp://0.0.0.0:26657"
            }
        },
        "sync_info":{
            "latest_block_hash":"5CD20D6553DDBE078EB43AC970F6D391F8FDFB2D2BD968F7B1A7454AA05154C5",
            "latest_app_hash":"847D19668188D482BFAA8808FB6207315F0766098A9202103F33DF51070027A5",
            "latest_block_height":"73830",
            "latest_block_time":"2024-06-04T15:38:53.899045777Z","earliest_block_hash":"EACF6AFE38ABB21E97078359EFAAD7E61D0417BBBA2C7724BB042D38F73693E1","earliest_app_hash":"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
            "earliest_block_height":"1",
            "earliest_block_time":"2024-06-03T17:59:53.991818534Z",
            "catching_up":false
        },
        "validator_info":{
            "address":"BEBEAE312EEE6AAD0E315FC6A36C9C44BDCEA1D3",
            "pub_key":{"type":"tendermint/PubKeyEd25519","value":"0/8w6RR2ZiQLwdbn1QvwrxYSfGUtW7jXTyPOL9a9NiY="},"voting_power":"100000"
        }
    }
}

Use the node_info.id value to set TELLORNODE_ID variable with NO QUOTES

Set the node name and moniker used for your node. Can be whatever name you want it to be
    - will be used to name the folder that your node config is in. Should be in the default location of already set in the LAYERD_NODE_HOME variable as "$HOME/.layer/$NODE_NAME"

run "sh ./layer_scripts/{selected version of script}.sh" inside of the layer base folder and it should set things up and start the chain thus making your node start syncing with your seed/peer


Creating a validator:

Assumes that you have a running and synced up node already

1. Set variable values
    - Use the same values you used for creating your node. Then select the amount of trb you would like to receive to from the faucet and stake for the new validator (1 TRB == 1e1*6 loya)

2. Run script from layer base folder with command: sh ./layer_scripts/create_new_validator_{OS}.sh 

3. After you have created your keys and made the transaction to create a new validator watch the output to ensure the returned validator info shows "status": 3
    - if status != 3 then cancel the script with CTRL-C

4. Whenever the script tells you to go to the terminal or screen session that your current node is running and stop the chain using CTRL-C. This will allow for the create validator script to restart the chain/node but this time it will run as a validator



Reporting Data:

Assuming you have already configured layer and have an account/address

1. Go to your layer directory or whereever you have access to the "./layerd" command

2. Run "./layerd query staking validators --node=http://tellornode.com:26657" to get the output of the current validators and pick which one you would like to delegate to from the operator address field (look for one with a status of 3, which means it is currently in a BONDED state)

3. Run "./layerd query bank balances {your "tellor" prefixed address} --node=http://tellornode.com:26657" to know the amount of loya you have.

4. Run "./layerd tx staking delegate {operator address of validator} {amount to delegate in loya} --gas auto --keyring-backend test --home ~/.layer/{NODE_NAME} --from {your address} --node=http://tellornode.com:26657 --chain-id layer" this will delegate to a reporter and allow you to create a validator. Please note that how much you delegate will also be how much your reporter power is

5. Run "./layerd tx reporter create-reporter {commission rate (ex: 200)} {minimum tokens required for someone to delegate to you (ex: 1000000)} --gas auto --keyring-backend test --keyring-dir ~/.layer/{NODE_NAME} --from {your address} --node=http://tellornode.com:26657 --chain-id layer". This will create your reporter and allow you to submit reports.

6. In order to submit a report you can either call the submit-value tx (./layerd tx oracle submit-value [creator] [qdata] [value] [salt] [flags]) or the commit-report tx (./layerd tx oracle commit-report [creator] [query_data] [hash] [flags])if you want to hide your value until the reveal window

Example of submit-value tx for trb-usd spot price:
./layerd tx oracle submit-value tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l 00000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000953706f745072696365000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000003747262000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000037573640000000000000000000000000000000000000000000000000000000000 0000000000000000000000000000000000000000000000000000000004a5ba50 0x00 --gas auto --keyring-backend test --keyring-dir ~/.layer/reporter --from tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l --node=http://tellornode.com:26657 --chain-id layer

Creating New Data Spec:

1. To create a new data spec you should first go look at our dataspec repo (https://github.com/tellor-io/dataSpecs) for an example of what they look like and make one for your new spec.

2. After you have filled out the template, you will need to run the following command using the info for your spec:

DATA_SPEC_JSON=$(cat <<EOF
{
    "document_hash": "IPFS hash of your dataspec file",
    "response_value_type": "bytes",
    "abi_components": [{
        "name": "chainId",
        "field_type": "uint256",
    }, {
        "name": "contractAddress",
        "field_type": "address",
    }, {
        "name": "calldata",
        "field_type": "bytes",
    }],
    "aggregation_mothod": "WeightedMode",
    "registrar": "tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l",
    "report_buffer_window": "300s",
}
EOF
)

echo $DATA_SPEC_JSON >> data_spec_json.json

This command will create the json object needed to pass into the ./layerd command. Please ensure that all of the information is correct and changed to your data spec.

3. Once the json file is created and you have checked the info in the resulting file looks correct. Run the following command...

./layerd tx registry register-spec {spec name} {Location of json file (ex: ./data_spec_json.json)} --keyring-backend test --keyring-dir ~/.layer/reporter --from {your tellor address} --yes --chain-id layer --node=http://tellornode.com:26657 --gas auto


Creating a Dispute:

./layerd tx dispute propose-dispute [report] [dispute-category] [fee] [pay-from-bond] [flags]

1. If you want to make a dispute made by a reporter you will need to know already or query the report object that you would like to propose a dispute for

    ./layerd query oracle get-reportsby-reporter  {address of reporter who made the report} --node=http://tellornode.com:26657

        *** this will return the reports made by the reporter but you will need to make sure that it is the report you are wanting to dispute if there is more than 1

    ./layerd query oracle get-reportsby-reporter-qid tellor19qg37zec70mzm9grhfp37rquk7hu89sldz2v4l XBPNnJfbuY8kKcEBoqgVDmx6Ddr/YSTuF2o6QRBn3tA= --node=http://tellornode.com:26657



