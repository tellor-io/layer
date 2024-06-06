Requirements:
    - go@1.22
    - sed
    - jq

If you are planning on not using the default port numbers (26656, 26657, 9090, 1317, etc) then make sure you update the script to point to the desired port numbers

start_one_node.sh (has been run solely on macs)
    - this script starts the chain with one node and validator
    - Unless you want to change the name of the folder used all you have to do to run this script is "sh ./start_scripts/start_one_node.sh and it will spin it up with the config being set up at ~/.layer/alice and the name of your keys/node is alice


start_one_node_aws.sh (has been run solely on ubuntu ec2 instances)
    - this script starts the chain with one node and validator and allows you to import a faucet account with a seed phrase.
    - Unless you want to change the name of the folder used all you have to do to run this script is "sh ./start_scripts/start_one_node_aws.sh and it will spin it up with the config being set up at ~/.layer/alice and the name of your keys/node is alice


