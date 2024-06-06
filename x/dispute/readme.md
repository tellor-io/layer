# `x/dispute`

## Abstract

This module handles disputes. Disputes can be handles in 3 levels: as a warning, minor infraction, or major infraction. For more information, reference the [ADRs](#adrs) below.

## ADRs

- adr002 - queryId time frame structure
- adr1002 - dual delegation
- adr1006 - dispute levels
- adr1007 - usage of staked tokens for disputes
- adr1008 - voting power by group
- adr2001 - trb bridge structure

## Transactions

- `ExecuteDispute`
- `AddFeeToDispute`
- `ProposeDispute`
- `Vote`
- `TallyVote`
- `UpdateTeam`

## Getters

`Params` - get module parameters

## Mocks

1. cd into registry/mocks
2. run `make mock-gen`

## CLI

### Example Commands