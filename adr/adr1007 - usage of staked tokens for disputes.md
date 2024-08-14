# ADR 1007: Usage of Staked Tokens for Disputes

## Authors

@themandalore 
@brendaloya

## Changelog

- 2024-02-22: initial version
- 2024-04-01: expanded on discussion
- 2024-08-03: clean up

## Context

In tellor's dual delegation model, the question arises who can use tokens to initiate the dispute?  Can the validator or reporter use tokens delegated to them? Can staked tokens even be used?

The current implementation is that only reporters can use their delegated tokens for disputes.  Not only does this make the code more straightforward, but it helps to separate the tasks - validators do chain stuff, reporters are in charge of whether the data is accurate.  

## Alternative Approaches

### Only the validator can use them for a dispute

The option to allow the validator to use the staked tokens to initiate a dispute would not be consistent with the separation of duties between the validators and reporters. 

### Both reporter and validator can use them for a dispute

We had thought about allowing the validator and reporter to use the staked tokens, but we don't want disputes to be a race between validators and reporters.  If validators and reporters are racing to start a dispute, it could be seen as a negative where they're not thinking out the situation fully and validators could decide to censor the reporter's transaction to give preference to their dispute transaction.

### Neither can use them for a dispute (only free floating tokens)

Old tellor does not allow staked tokens to be used for disputes.  This option works well in the scenario when the percentage of staked tokens is less than 50% of the circulating supply, which is the case in the current system. It also incentivizes users to hold free floating tokens to initiate and vote on disputes and token holders to keep tokens off CEXs for the possibility of earning a profit from successful disputes. Currently, being a profitable reporter on Tellor requires high technical skills. 

However, in Tellor Layer delegation will make it easier for token holders to earn a profit without technical skills. Assuming a scenario where a high (>50%) percent of tokens are staked, starting a major dispute with only free floating tokens could become infeasible.  Consensus assumes that up to 1/3 could be compromised.  If we assume that 1/3 of staked tokens is malicious, we need to have that amount of tokens available to dispute in the case of an attack.  If those tokens are all locked on CEXs or not free floating and ready to be used for disputes, there could be a costly delay. By allowing parties to initiate a dispute with staked tokens, there will always be enough tokens for disputes on Layer as long as the consensus 50% is not breached.  

## Issues / Notes on Implementation


