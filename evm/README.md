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

Then install dependencies from this `evm` dir:
```bash
npm i
```

Create a `secrets.yaml` file in the root layer directory:
```
touch secrets.yaml
```
And set your alchemy eth-mainnet key:
```yaml
eth_api_key: "your-alchemy-eth-mainnet-key"
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


### TokenBridge-FunctionTests with Live chain

Note that the withdrawFromLayer test will fail for now without Layer time fast-forwarding. 

```bash
npx hardhat test fullTest/TokenBridge-FunctionTests.js
```

But you can request to withdraw tokens from layer by running this from the layer home dir:

```./layerd tx bridge withdraw-tokens 88dF592F8eb5D7Bd38bFeF7dEb0fBc02cf3778a0 100loya --from charlie --chain-id layer --home ~/.layer/alice
```

### Live chain bridge-TestsAuto

Make sure the hardhat forking network is commented out in `hardhat.config.js`, and the normal `hardhat: {}` network is uncommented.

Then run the test:
```bash
npx hardhat test fullTest/Bridge-TestsAuto.js
```
All tests should pass except for the "optimistic value" test, but you should see instructions printed out for how to request new attestations for a past value. Run the layer command to request a new attestation:
```bash
"./layerd tx bridge request-attestations {query-id} {timestamp} --from charlie --chain-id layer --home ~/.layer/alice"
```

Update the `PAST_REPORT_TS` variable in the test file, and re-run the tests. 

