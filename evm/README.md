# Layer EVM
This directory holds the EVM side of Layer.

## Running tests

### Hardhat Function Tests

```
npx hardhat test
```

### Live chain tests - Setup

To run tests, create your `.env` file in this `evm` dir:
```bash
cp .env.example .env
```

Then add your node url to `.env`:
```
NODE_URL="your-node-url"
```

Install dependencies from this `evm` dir:
```bash
npm i
```

Create a `secrets.yaml` file in the root layer directory:
```
touch secrets.yaml
```
And set your eth-mainnet rpc url:
```yaml
eth_rpc_url: "your-eth-mainnet-rpc-url"
```

To run the start scripts for the layer chain, you need `jq` installed. For MacOS, you can install it with 
```bash
brew install jq
```
Or on Ubuntu, 
```bash
sudo apt-get install jq
```

Comment out the following lines in `x/reporter/ante/ante.go`, lines 34-35 at the time of writing. This will allow you to run a second validator which changes the validator power by more than 5%.
```go
// case *stakingtypes.MsgCreateValidator:
//     msgAmount = msg.Value.Amount
```

In `terminal-1`, go to the layer root directory and run:
```bash
chmod 755 ./start_scripts/start_two_chains.sh
chmod 755 ./start_scripts/start_bill.sh
./start_scripts/start_two_chains.sh
```

In `terminal-2`, run:
```bash
./start_scripts/start_bill.sh
```

In `terminal-3`, create two reporters:
```bash
./layerd tx reporter create-reporter "100000000000000000" "1000000" --from alice --keyring-backend test --chain-id layertest-1 --home ~/.layer/alice --keyring-dir ~/.layer/alice --fees 1000loya --yes
```
and for bill:
```bash
./layerd tx reporter create-reporter "100000000000000000" "1000000" --from bill --keyring-backend test --chain-id layertest-1 --home ~/.layer/bill --keyring-dir ~/.layer/bill --fees 1000loya --yes
```

### TokenBridge-FunctionTests with Live chain

Note that the withdrawFromLayer test will be skipped for now without Layer time fast-forwarding. 

```bash
npx hardhat test fullTest/TokenBridge-FunctionTests.js
```

But you can request to withdraw tokens from layer by running this from the layer home dir:

```bash
charlies_address=$(./layerd keys show charlie --home ~/.layer/alice -a)
./layerd tx bridge withdraw-tokens $charlies_address 88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0 100loya --from $charlies_address --chain-id layertest-1 --home ~/.layer/alice --keyring-backend test --keyring-dir ~/.layer/alice --fees 500loya --yes
```

### Live chain bridge-TestsAuto

Run the tests:
```bash
npx hardhat test fullTest/Bridge-TestsAuto.js
```
All tests should pass except for the "optimistic value" test, but you should see a timestamp printed out. Use this timestamp in place of {timestamp} in the command below to request new attestations:
```bash
charlies_address=$(./layerd keys show charlie --home ~/.layer/alice -a)
./layerd tx bridge request-attestations $charlies_address 83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce4992 {timestamp} --from charlie --chain-id layertest-1 --keyring-backend test --keyring-dir ~/.layer/alice --fees 1000loya --yes
```

Update the `PAST_REPORT_TS` variable in the test file with the same timestamp you used above, and re-run the tests. 

