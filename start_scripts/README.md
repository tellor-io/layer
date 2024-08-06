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


Logging:

If you want to record only the failed transactions into a different log file do the following stops

1. create a new terminal or screen instance and call mkfifo mypipe inside of the layer directory

2. Then from that terminal call "./log_filter_failed_tx.sh". This will just start a script and not log anything (it will block you from doing anything else in this terminal instance going forward until you are done).

3. Create another terminal instance and run the following command: tail -f fulldata_first_node_logs.txt >> mypipe &

    This command will read in lines from the full log file as they are written and will send them to our pipe script running in the other terminal instance where it will filter all logs coming in for the failed transaction string. It will output something like ([1] 53386) and nothing else but don't worry the pipe is still going and collecting logs

