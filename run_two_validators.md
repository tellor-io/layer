# Cosmos Multiple Validators Tutorial

## 1. Prerequisites

```
Ignite CLI
```

## 2. Initialize your chain

Create a binary in your terminal:

```
ignite scaffold chain <application-name>
```

Build the application chain:

```
cd <application-name>
ignite chain build
```

## 3. Create keys for your nodes

Cosmos SDK will automatically create a binary cli, and we can use it as a command: `appd`
Please make sure to check your default keyring on client.toml. 
If it is "os", the folder "keyring" will not show up.
```
appd keys add <key-name-1>
appd keys add <key-name-2>
...
```

## 4. Initialize chain nodes
Here you will init multiple folders for different nodes.
Note here: Two nodes have to have different config and data folder.
```
appd init <moniker1> --home <addr-1> --chain-id <your-chain-id>
appd init <moniker2> --home <addr-2> --chain-id <your-chain-id>
...
```

## 5. Add genesis accounts
Add genesis for main chain, other chain do not have role to do this.
Instead, they will sync with the first chain on below instruction
```
appd add-genesis-account $(appd keys show <key-name-1> -a) <amount>stake --home <addr-1>
appd add-genesis-account $(appd keys show <key-name-2> -a) <amount>stake --home <addr-1>
...
```

## 6. Stake into the chain

```
appd gentx <key-name-1> <amount>stake --chain-id <your-chain-id> --home <addr-1>
appd collect-gentxs --home <addr-1>
```

## 7. Start the chain node

Before you start, the `api` server and `swagger` should be enabled. You can go to the `app.toml` to change these. Then:

```
appd start --home <addr-1>
```

## 8. Config the next chain node to sync with the blockchain

Go to the folders that contain the `config` and `data` directory of the chains:

```
cd <addr-2>
```

Copy these files of the first chain into the `config` folder:

- `genesis.json`
- `app.toml`
- `client.toml`
- `config.toml`

Change every ports in these 3 `.toml` files to differ from the first chain.

You also need to add the `seeds` and the `persistent_peers`:

```
seeds="<node-id-1>@<listen-addr>:<port>,<node-id-1>@<listen-addr>:<port>"
persistent_peers="<node-id-1>@<listen-addr>:<port>,<node-id-1>@<listen-addr>:<port>"

```

To get the node id, run this command:

```
appd tendermint show-node-id
```

After everything is done, you can start the node with:

```
appd start <addr-2>
```

NOTE: in local development, if using `127.0.0.1 and the node throws an error, you can try `localhost` instead.

## 9. Run the node as a validator

Before sending the create-validator transaction, you need to send some tokens from the existing validator to the new ones. Run this command:

```
appd q account <account-addr> --chain-id testnet --home <home-dir>
```

You might see an error is thrown saying that the account is not found. This problem may occur if the account was created on a different node or on a different chain with a different chain ID. When querying for an account, the node will look for the account's address in its local state database. If the account was not created on that node or chain, then the node will not have the account's address in its local state database and will return an error.

To solve this problem, you can send tokens from the existing validator to the new ones, so that the validator can recognize your new account address. Run this command:

```
appd tx bank send <account-addr> <amount>stake --chain-id testnet --home <home-dir>
```

Now you can run the previous command to check the balance of the account.

Create a staking transaction by running:

```
appd tx staking create-validator \                          
  --amount=<amount>stake \
  --pubkey=$(appd tendermint show-validator --home <addr-2>) \
  --moniker=<moniker2> \
  --chain-id=<your-chain-id> \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1000000" \
  --gas="auto" \
  --from=<key-name-2> --home <addr-2>
```

Restart the node, and you will have another validator.

You can check the status of all validators with:

```
appd tendermint show-validator --home <addr-2>
```

or with:

```
appd q staking validator cosmosvaloper1yourvalidatoraddress --chain-id=testnet --node=tcp://127.0.0.1:26657
```

If the status of the validator is `BOND_STATUS_UNBONDED`, you might need to delegate more tokens, you can run:

```
appd tx staking delegate <account-addr> <amount>stake --chain-id testnet --home <home-dir>
```

If the status of the validator is `BOND_STATUS_BONDED`, then congrats, you have a new validator for your network.
