Once the chain is running please select an rpc url of a current node to use in your set up of your node and set the LAYER_NODE_URL variable to that url (ex: tellor-example-node.com) with no quotes

After finding a node url to use call "curl NODE_URL:26657/status" this should return something like this:

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

use the node_info.id value to set TELLORNODE_ID variable

Set the node name and moniker used for your node 

run "sh ./layer_scripts/{selected version of script}.sh" inside of the layer base folder and it should set things up and start the chain thus making your node start syncing with your seed/peer

