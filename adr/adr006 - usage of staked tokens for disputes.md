# ADR 006: Usage of Staked Tokens for Disputes

## Authors

@themandalore

## Changelog

- 2024-02-22: initial version

## Context

In tellor's dual delegation model, the question arises who can use tokens to initiate the dispute?  Can the validator or reporter use tokens delegated to them?  Can staked tokens even be used?

The current implementation is that only reporters can use the ones delegated for disputes.  Not only does this make the code more straightforward, but it helps to separate the tasks - validators do chain stuff, reporters are in charge of whether the data is accurate.  


## Alternative Approaches

### Only the validator can use them for a dispute

This option would also work.  More for us, just keeping it consistent with the separation of duties. 

### both reporter and validator can use them for a dispute

We had thought about this, but we don't want it to be too much of a race.  If validators and reporters are racing to start a dispute, it could be seen as a negative where they're not thinking out the situation fully. 

### Neither can use them for a dispute (only free floating tokens)

This is how old tellor works.  This option is fine, but in a world where we hope that a high (>50%) of tokens are staked, starting a major dispute could become infeasible.  Consensus assumes that up to 1/3 could be compromised.  If we assume that 1/3 of staked tokens is malicous, we need to have that amount of tokens available to dispute in the case of an attack.  If those tokens are all locked on CEX's or not just waiting to dispute, there could be a costly delay to the dispute.  By allowing staked parties to initiate the dispute, you will always have enough tokens as long as the consensus 50% is not breached.  

## Issues / Notes on Implementation

