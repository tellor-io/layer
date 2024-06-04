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

