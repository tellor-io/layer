## statesync.sh

Use statesync.sh to configure (or reconfigure) state sync for a tellor layer node.

***Please backup your home directory before using any scripts on it!***

To use a statesync script: 

1) Download the script and move it to the same directory as your layerd binary.

2) Give the script permission to execute:

```
chmod +x statesync-tellorlayer.sh
```
3) Run the script:

```
./statesync-tellorlayer.sh
```

Sample output:

```
This script will clear all chain data from your local layer node and resync the chain.
Your configurations and accounts will be preserved!
Press enter to continue or ctrl+c to exit
A2E9B7CE289109A5788FD1CBB3952DCED32F266D8FADE6D980BDC3DF2FFFDAAA


Node URL: https://node-palmito.tellorlayer.com/rpc/
Node ID: ac7c10dc3de67c4394271c564671eeed4ac6f0e0
Current height: 3843769
Likely snapshot height: 3822500
Trusted hash: A2E9B7CE289109A5788FD1CBB3952DCED32F266D8FADE6D980BDC3DF2FFFDAAA


Press enter to continue or ctrl+c to exit
```