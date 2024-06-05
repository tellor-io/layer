# `x/oracle`

## Abstract

This module 

## ADRs

adr

## Transactions

`MsgCommitReport`
`MsgSubmitValue`
`MsgTip`
`MsgUpdateCycleList`

## Getters

`Params` - get module parameters
`Reporters` - get all staked reporters
`DelegatorReporter` - get reporter a delegator is staked with.
`ReporterStake` - get a reporter's total tokens.
`DelegationRewards` - get rewards of a delegator
`ReporterOutstandingRewards` - get all outstanding rewards for a reporter

## Mocks

1. cd into oracle/mocks
2. run `make mock-gen`

## CLI

### Example Commands
