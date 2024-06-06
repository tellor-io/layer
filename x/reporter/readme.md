# `x/reporter`

## Abstract

This module enables token holders to stake tokens to become data reporters and delegate to other reporters. For more information about how reporting works, reference the [ADRs](#adrs) below.

## ADRs

adr1001 - distribution of base rewards
adr1002 - dual delegation
adr1005 - handling of tips after report
adr1008 - voting power by group
adr2001 - trb bridge structure

## Transactions

`CreateReporter`
`ChangeReporter`
`UnjailReporter`
`WithdrawTip`
`UpdateParams`

## Getters

Params - get module parameters
Reporters - get all staked reporters
DelegatorReporter - get reporter a delegator is staked with.

## Mocks

1. cd into registry/mocks
2. run `make mock-gen`

## CLI

### Example Commands