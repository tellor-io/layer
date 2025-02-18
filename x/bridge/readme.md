# `x/bridge`

## Abstract

This module enables Layer to communicate with user chains. It enables both token bridging and data bridging. For more information, reference the [ADRs](#adrs) below.

## ADRs

- [adr001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr001.md) - chain size limitations  
- [adr2001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2001.md) - trb bridge structure  
- [adr2002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2002.md) - nonces for bridging  
- [adr2003](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2003.md) - vote extensions vs txns for bridge  
- [adr2006](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2006.md) - data bridge architecture  
- [adr3001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr3001.md) - TellorMaster to bridge structure

## Tx

### ClaimDeposits
Claim deposits made by the Ethereum token bridge contract into Layer.  
- `./layerd tx bridge claim-deposits [creator] [deposit-ids] [timestamps]`

- `./layerd tx bridge claim-deposits tellor1p88ju0yhutmf5p2u798xv3umaa7ujw7gch9r4f 27 1713024000`

### RequestAttestations
Request attestations for a snapshot of an aggregate report.  
- `./layerd tx bridge request-attestations [creator] [query-id] [timestamp]`

- `./layerd tx bridge request-attestations tellor1p88ju0yhutmf5p2u798xv3umaa7ujw7gch9r4f 3375a5d1a012c725a51f641a86a09e37627ec21ec907401e9b95f7d1ecd22af6 1713024000`

### WithdrawTokens
Withdraw tokens from Layer to the recipient address through the token bridge contract.  
- `./layerd tx bridge withdraw-tokens [creator] [recipient] [amount]`

- `./layerd tx bridge withdraw-tokens tellor1p88ju0yhutmf5p2u798xv3umaa7ujw7gch9r4f AE7CFe4CF579Ec060f95d951bD5260A5A8c0dcDC 1000000loya --fees 10loya --chain-id layertest-3`

### UpdateSnapshotLimit
Governance transaction to update the number of attestation requests per block.  
- `./layerd tx bridge update-snapshot-limit [limit]`

## Queries

### Params
- `./layerd query bridge params`

### GetAttestationDataBySnapshot
- `./layerd query bridge get-attestation-data-by-snapshot [snapshot]`

### GetAttestationsBySnapshot
- `./layerd query bridge get-attestation-by-snapshot [snapshot]`

### GetEVMAddressByValidatorAddress
- `./layerd query bridge get-evm-address-by-validator-address [validator-address]`

### GetEVMValidators
- `./layerd query bridge get-evm-validators`

### GetSnapshotsByReport
- `./layerd query bridge get-snapshots-by-report [query-id] [timestamp]`

### GetValidatorCheckpointParams
- `./layerd query bridge get-validator-checkpoint-params [timestamp]`

### GetValidatorCheckpoint
- `./layerd query bridge get-validator-checkpoint`

### GetValidatorTimestampByIndex
- `./layerd query bridge get-validator-timestamp-by-index [index]`

### GetValsetByTimestamp
-  `./layerd query bridge get-valset-by-timestamp [timestamp]`

### GetValsetSigs
- `./layerd query bridge get-valset-sigs [timestamp]`

### GetSnapshotLimit
- `./layerd query bridge get-snapshot-limit`

## EndBlock
### CompareAndSetBridgeValidators
 Function for loading last saved bridge validator set and comparing it to current set. If the validator set power has changed by more than 5% or the validator set is stale, it updates the bridge validator set and validator params.

### CreateNewReportSnapshots
Creates new report snapshots for all reports aggregated at the current height.

## Mocks

`make mock-gen-bridge`