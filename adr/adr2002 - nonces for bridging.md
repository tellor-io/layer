# ADR 2002: Nonces for Bridging

## Authors

@themandalore
@tkernell
@brendaloya

## Changelog

- 2024-02-21: initial version
- 2024-02-26: unbonding period 
- 2024-04-04: clarity in background
- 2024-08-04: clean up

## Context

Nonces can help keep track of the order that data is added to Layer and the validator updates to the bridge. Keeping track of this order is important because allowing users to request old data can create optionality in the data used in other protocols (e.g. allowing users to request old price data for their benefit instead of the latest available). This raised the question, should Layer allow older signed data to be pushed after newer signed data? To make it transparent (or not allow it) when this is happening, nonces are needed to keep track of when data is created. 

The order of the validator set updates is also very important because it helps maintain a list of validators that are still bonded and active. While blobstream has used nonces to track the order of these important pieces, in Layer we have opted to use timestamps as described below for oracle data attestations and validator set changes. 

### Data attestations

We do not include a nonce for oracle data attestations. We use attestation timestamps and report timestamps instead to determine the ages of these, respectively. We do not necessarily need every single oracle attestation to be bridged to every single user chain, and they do not have to be bridged in the order in which they were created. We include `nextTimestamp` and `previousTimestamp` in each oracle attestation to determine ordering properties of oracle reports.

### Validator set changes

However, instead of a universal nonce, Layer includes a timestamp as one of the parameters used in bridge validator set updates. This is different than the original blobstream which also had a universal nonce. Each time the bridge validator set changes, all validators sign a hash of multiple parameters, including the new validator set timestamp. The bridge contract enforces that this timestamp is greater than the previous one. The bridge contract also uses this timestamp to determine whether a given validator set record is stale by comparing the timestamp to the Layer unbonding period. We use the strictly increasing timestamp rule to prevent the bridge from accepting old validator set attestations, and it also allows the bridge to save gas costs by skipping blocks. In addition, we use the age of a validator set attestation to determine whether that validator set's tokens are still locked as stakes and thus eligible for slashing.


## Alternative Approaches

### Require a universal nonce

We considered implementing a universal nonce but bacause that would mean that every time data is signed on layer it would need to be pushed across the bridge.  This is beneficial for DA systems (like Celestia's blobstream) since these need blocks to be processed in order.  If not, a situation could emerge where some data can front run other data and prevent it from being pushed. For our purposes, we only require that timestamps move forward in terms of the block signed. This means that some updates on layer can be skipped over (e.g. if we report prices 100, 101, and 102, a user on ETH could just bridge the 102 update and skip the other two). If a user wants to preserve the order of data pushed to Tellor, they can implement a nonce in the query and force increase it per request.  


## Issues / Notes on Implementation

TODO: explain how this is related to the nonce/disputes and valid set of validator set
We had originally wanted the validator to be staked until unbonded (e.g. 21 day unbonding period), but a validator set can change significantly very quickly due to disputes and slashing events, so comparing a validator set's age to the unbonding period does not ensure that a given validator still has tokens at stake. For this reason, we use the validator nonce and don't allow them to keep reporting if the validator set changes quickly.

## Links

https://github.com/celestiaorg/blobstream-contracts 