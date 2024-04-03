# ADR 1004: Fees on Tips

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-21: initial version
- 2024-04-01: 

## Context

A fee on tips is necessary to discourage vote farming and spaming the network by tippig data that has no support. 

The 2% fee on tips was chosen arbitrarily. 


## Alternative Approaches

### No Fee
Implemeting no fee has been discarted since the old version of tellor because it can encorage vote farming. In Layer, reporters will not earn voting weight per report count, however, users will so a fee is still necessary. 

### Higher fee

We could just increase that 2% fee to something like 5 or 10%.  Although this could be a solution, limiting that fee is important because it supports having on-chain tips (can be tracked for governance/ usage purposes) vs off-chain.  


## Issues / Notes on Implementation

The fee can be increased if vote farming or spamming is observed. 