# `x/reporter`

## Abstract

This module 

## ADRs

adr

## Transactions

`MsgCreateReporter`
`MsgDelegateReporter`
`UndelegateReporter`
`WithdrawReporterCommission`
`WithdrawDelegatorReward`

## Getters

Params - get module parameters
Reporter - get a reporter by address
Reporters - get all staked reporters
DelegatorReporter - get reporter a delegator is staked with.
ReporterStake - get a reporter's total tokens.
DelegationRewards - get rewards of a delegator
ReporterOutstandingRewards - get all outstanding rewards for a reporter
ReporterCommission - get a reporter's commission reward

## Mocks

1. cd into registry/mocks
2. run `make mock-gen`

## CLI

### Example Commands