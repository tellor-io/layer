# ADR 2012: Domain separator and testnet upgrade
## Authors

@themandalore

## Changelog

- 2024-10-1: initial version


## Context

Validators sign attestations for reports and bridge validator set updates.  Unfortunately, the first version of layer had validators sign the messages the same irregardless of chain-id (testnet or mainnet).  This meant that signed attestations on testnet could be used as a signature on mainnet.  Since the validator sets would be (are) different, the signing party could be slashed for signing a "malicious" attestation (e.g. trying to update to an incorrect validator set). Additionally, the more serious concern would be that the mainnet databridge contract could get tricked into accepting testnet attestations if over 2/3 of mainnet validators are also participating on testnet with the same keys.  Originally it was thought that parties would not use the same keys for tesnet and mainnet and thus not be a problem, but the mythical nature of this belief was quickly realized. 

To do the fix, we updated domain separator to be different between networks to ensure that no one can get slashed using their attestations on the other chain whether it be testnet or mainnet.  To explain, the attestation signatures include a fixed character called the VALIDATOR_SET_HASH_DOMAIN_SEPARATOR that is appended to details we want to sign (e.g. the power threshold, timestamp, and hash of the list of validators).  To differentiate from mainnet, we changed this fixed value for testnet to be the hash of the chain_id and "checkpoint" (keccak256(abi.encode("checkpoint", TELLOR_CHAIN_ID))). This allows for multiple testnets and the mainnet to all have different attestations.  

The fix required us to redeploy the dataBridge and also tokenBridge as a result on Sepolia testnet.  Since it is testnet, the team has ownerhip of the "deity" key for the sytem, allowing us to update the proxy contract which the token address reads from (we cannot do this on mainnet).  This meant that we could update the token rewards to go to the correct token bridge by implementing a variable change via a proxy upgrade.  You can see it in this script [https://github.com/tellor-io/layer/blob/main/evm/scripts/UpgradeSepolia.js](https://github.com/tellor-io/layer/blob/main/evm/scripts/UpgradeSepolia.js)

## Alternative Approaches

An initial quick fix was to just prevent validators from using the same key on testnet and mainnet.  This was unfortunately just a stop gap though as the behavior could lead to slashing, so our ability to monitor should not be the feature preventing a malicious attestation.  

## Issues / Notes on Implementation

When upgrading, we did hit the issue of initializing the data bridge validator set before the chain upgrade.  Since the chain upgrade had not happened yet, the checklist was signed with the wrong domain separator, forcing us to redeploy the data bridge.  It was fixed shortly after.  

## Links


