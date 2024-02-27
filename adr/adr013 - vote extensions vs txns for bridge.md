# ADR 013: vote extensions vs transactions for bridge signatures

## Authors

@themandalore

## Changelog

- 2024-02-27: initial version

## Context

In order to parse validator signatures on EVM chains, tellor validators need to sign oracle data with ecsda keys (not the normal edd keys used in tendermint).  There are two places where we can place these signatures, either as transactions on the chain, or as vote extensions.  Vote extensions in the cosmos sdk, are simply data that is appended to validator signatures on a given block.  Since it is required that validators commit this data, adding them as vote extensions makes sense as it will force validators to mantain the bridge.  The limit to vote extensions is that the data must be very limited in size.  Too many signatures could result in the chain slowing down (or bridge information being ommited).  Additionally, vote extensions are a relatively newer feature of the sdk.  


## Alternative Approaches

### place as txns

The original idea was to just have validators submit a transaction in each block after finalized oracle data.  The issue with this approach is that the proposer for any given block has control over what transactions get included.  This means that they could censor signatures from certain validators and it would impossible to tell whether they were censored or failed to submit the transaction.  Additionally, size issues still exist with transactions, and storing each signature as a transaction is much larger on aggregate (when considering chain state size growth).  The transaction method would also force validators to pay gas on signature transactions and compete for space in each block with non-bridge signature transactions (e.g. data submissions). 

### use zk method, no signatures

A future option (that celestia is also taking) is to completely abandon external signatures and opt for zk methods.  I would say that this is the long term plan, however current zk methods are so novel that relying on them would be more akin to experimentation than actual robust usage.  Additionally, proving times for most of these methods is still prohibitively slow for many oracle use cases and may also add a centralization vector if advanced hardware is required.  


## Issues / Notes on Implementation

