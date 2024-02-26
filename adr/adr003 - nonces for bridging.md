# ADR 003: Nonces for Bridging

## Authors

@themandalore
@tkernell

## Changelog

- 2024-02-26: unbonding period 
- 2024-02-21: initial version

## Context

We include a timestamp as one of the parameters used in bridge validator set updates in place of a nonce. This is separate than the original blobstream which also had a univesal nonce. Each time the bridge validator set changes, all validators sign a hash of multiple parameters, including the new validator set timestamp. The bridge contract enforces that this timestamp is greater than the previous one. The bridge contract also uses this timestamp to determine whether a given validator set record is stale by comparing the timestamp to the Layer unbonding period. We use the stricly increasing timestamp rule to prevent the bridge from accepting old validator set attestations, and it also allows the bridge to save gas costs by skipping blocks. In addition, we use the age of a validator set attestation to determine whether that validator set's tokens are still locked as stakes and thus eligible for slashing.

We do not include a nonce for oracle data attestations. We use attestation timestamps and report timestamps instead to determine the ages of these, respectively. We do not necessarily need every single oracle attestation to be bridged to every single user chain, and they do not have to be bridged in the order in which they were created. We include `nextTimestamp` and `previousTimestamp` in each oracle attestion to determine ordering properties of oracle reports.


## Alternative Approaches

### require a universal nonce

This requires pushing of each request.


## Issues / Notes on Implementation

The validator set can change significantly very quickly due to disputes and slashing events, so comparing a validator set's age to the unbonding period does not ensure that a given validator still has tokens at stake. 

## Links

https://github.com/celestiaorg/blobstream-contracts 