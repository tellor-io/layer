# ADR 2002: Nonces for Bridging

## Authors

@themandalore
@tkernell

## Changelog

- 2024-02-21: initial version
- 2024-02-26: unbonding period 
- 2024-04-04: clarity in background


## Context

There is a need for nonces to keep track of when data is created on layer.  The problem is: should you allow older signed data to be pushed after newer signed data?  If you do, you can give optionality to bridge users for going back in time (e.g. a price feed). 

For layer include a timestamp as one of the parameters used in bridge validator set updates in place of a nonce. This is different than the original blobstream which also had a univesal nonce. Each time the bridge validator set changes, all validators sign a hash of multiple parameters, including the new validator set timestamp. The bridge contract enforces that this timestamp is greater than the previous one. The bridge contract also uses this timestamp to determine whether a given validator set record is stale by comparing the timestamp to the Layer unbonding period. We use the stricly increasing timestamp rule to prevent the bridge from accepting old validator set attestations, and it also allows the bridge to save gas costs by skipping blocks. In addition, we use the age of a validator set attestation to determine whether that validator set's tokens are still locked as stakes and thus eligible for slashing.

We do not include a nonce for oracle data attestations. We use attestation timestamps and report timestamps instead to determine the ages of these, respectively. We do not necessarily need every single oracle attestation to be bridged to every single user chain, and they do not have to be bridged in the order in which they were created. We include `nextTimestamp` and `previousTimestamp` in each oracle attestion to determine ordering properties of oracle reports.


## Alternative Approaches

### require a universal nonce

A universal nonce would mean that every time data is signed on layer, you would need to push it across to the bridge.  This is beneficial for DA systems (like Celestia's blobstream) as you want blocks to be processed in order.  If not, you could have a situation where some data can front run other data and prevent it from being pushed.  For our purposes, we only require that timestamps move foward in terms of the block signed.  This does mean that some updates on layer can be skipped over (e.g. if we report prices 100, 101, and 102, a user on ETH could just bridge the 102 update and skip the other two). If a user wants to preserve the order of data pushed to Tellor, they can implement a nonce in the query and force increase it per request.  


## Issues / Notes on Implementation

We had oringinally wanted to just say that a validator is staked until unbonded (e.g. 21 day unbonding period), but a validator set can change significantly very quickly due to disputes and slashing events, so comparing a validator set's age to the unbonding period does not ensure that a given validator still has tokens at stake. For this reason, we use the validator nonce and don't allow you to keep reporting if the validator set changes quickly.

## Links

https://github.com/celestiaorg/blobstream-contracts 