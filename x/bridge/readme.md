# `x/bridge`

## Abstract

This module enables Layer to communicate with user chains. It enables both token bridging and data bridging. For more information, reference the [ADRs](#adrs) below.

## ADRs

adr001 - chain size limitations
adr2001 - trb bridge structure
adr2002 - nonces for bridging
adr2003 - vote extensions vs txns for bridge
adr2006 - data bridge architecture
adr3001 - TellorMaster to bridge structure

## Transactions

`ClaimDeposit`
`RequestAttestations`
`WithdrawTokens`

## Getters

`Params` - get module parameters
`GetAttestationDataBySnapshot`
`GetAttestationBySnapshot`
`GetCurrentAggregateReport`
`GetDataBefore`
`GetEVMAddressByValidatorAddress`
`GetEVMValidators`
`GetSnapshotsByReport`
`GetValidatorCheckpointParams`
`GetValidatorCheckpoint`
`GetValidatorTimestampByIndex`
`GetValsetByTimestamp`
`GetValsetSigs`

## Mocks

1. cd into registry/mocks
2. run `make mock-gen`

## CLI

### Example Commands