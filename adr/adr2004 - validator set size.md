# ADR 2004: Validator Set Size

## Authors

@brendaloya
@themandalore

## Changelog

- 2024-02-21: initial version
- 2024-04-01: initial justification - sigs that fit in vote extension(max data + decentralization), rewards and currently observed # of validators
- 2024-04-04: additional context

## Context

The number of validators has direct effect on bridging. We have chosen 100 validator cap because it seems reasonable based on what we have observed in the current cosmos/tendermint ecosystem. Vote extension are relatively new to the cosmos sdk so the exact number to cap validators will be a function of the number of signatures we can fit in the vote extension(to maximize data bridging and decentralization), what we see currently in the cosmos/tendermint ecosystem, and how well that number of validators can be incentivized with the 25% validator rewards split (75% to reporters, 25% to validators).

Market conditions will take care of the economic incentives over time and reach equilibrium. However, maximizing the amount of data bridged with a sufficiently decentralized validator set may just be arbitrary initially and be adjusted via governance later.  The current plan is to work with the technology to increase the validator set size and reduce concentration where we can.  As bridge contracts get cheaper, zk methods become more readily available, and the cosmos sdk can handle more validators doing faster blocks, we plan to increase this number and decentralize as much as we can.  


## Alternative Approaches

### More validators

More validators, more decentralization, but the problem with more validators is that data needs to be shared across a larger set and can lead to slower blocks.  Additionally, each signature from validators must be parsed in an EVM smart contract for bridging.  A validator set of 100 can be parsed but is still rather expensive (~2M gas from initial estimates).  We can reduce this a little by only validating the largest ones (up to 2/3 power), however there is a direct tradeoff between validator decentralization and cost to verify. 

### Less validators

The inverse is of course true for less validators: cheaper bridging, faster blocktimes, but increased centralization.


## Issues / Notes on Implementation

