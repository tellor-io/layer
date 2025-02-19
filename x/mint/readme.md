# `x/mint`

## Abstract

This module enables the minting of time based rewards for reporters, validators, and the team. For more information about rewards and how the token bridge works, reference the [ADRs](#adrs) below.

## ADRs

- [adr1001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1001.md) - distributions of base rewards
- [adr1002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1002.md) - dual delgation
- [adr10003](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1003.md) - time based rewards eligibility
- [adr2001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2001.md) - trb bridge structure
- [adr3001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr3001.md) - TellorMaster to bridge structure

## Tx

### Init
Start minting time based rewards through a governance proposal.

## BeginBlocker
### MintBlockProvision
Mint the block provision for the current block.

### SetPreviousBlockTime
Sets the block time for the minter to read next block.

