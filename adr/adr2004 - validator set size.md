# ADR 2004: Validator Set Size

## Authors

@brendaloya
@themandalore

## Changelog

- 2024-02-21: initial version
- 2024-04-01: initial justification - sigs that fit in vote extention(max data + decentralization), rewards and currently observed # of validators

## Context

The number of validators has direct effect on bridging. We have chosen 100 validator cap because it seems reasonable based on what we have observed in the current cosmos/tendermint ecosystem. Vote extension are relatively new to the cosmos sdk so the exact number to cap validators will be a function of the number of signatures we can fit in the vote extension(to maximize data briging and decentralization), what we see currently in the cosmos/tendermint ecosystem, and how well that number of validators can be incetivized with the 25% validator rewards split (75% to reporters, 25% to validators).

Market conditions will take care of the economic incentives over time and reach equilibrium. However, maximizing the amount of data bridged with a sufficiently decentralized validator set may just be arbitrary initially and be adjusted via governance later.??? 


## Alternative Approaches

### More validators

More validators, more decentralization, but slower more expensive bridging.

### Less validators

Cheaper bridging, faster blocktimes, increased centralization. 


## Issues / Notes on Implementation

