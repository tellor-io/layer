# `x/oracle`

## Abstract

This module enables reporting of data to the network. Reporters can commit and then reveal their report, or immediatiely reveal. Reports can only be made for the cycle list and tipped queries. For more information, reference the [ADRs](#adrs) below.

## ADRs

- adr002 - queryId time frame structure
- adr1001 - distribution of base rewards
- adr1002 - dual delegation
- adr1003 - time based rewards eligibility
- adr1004 - fees on tips
- adr1005 - handling of tips after report
- adr1008 - voting power by group

## Transactions

- `SubmitValue`
- `Tip`
- `UpdateCycleList`

## Getters

- `Params` - get module parameters
- `Reporters` - get all staked reporters
- `DelegatorReporter` - get reporter a delegator is staked with.
- `ReporterStake` - get a reporter's total tokens.
- `DelegationRewards` - get rewards of a delegator
- `ReporterOutstandingRewards` - get all outstanding rewards for a reporter

## Mocks

1. cd into oracle/mocks
2. run `make mock-gen`

## CLI

### Example Commands
