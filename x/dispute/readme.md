# `x/dispute`

## Abstract

This module handles disputes. Disputes can be handles in 3 levels: as a warning, minor infraction, or major infraction. For more information, reference the [ADRs](#adrs) below.

## ADRs

- [adr002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr002.md) - queryId time frame structure  
- [adr1002](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1002.md) - dual delegation  
- [adr1006](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1006.md) - dispute levels  
- [adr1007](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1007.md) - usage of staked tokens for disputes  
- [adr1008](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr1008.md) - voting power by group  
- [adr2001](https://github.com/tellor-io/Layer/blob/main/docs/adr/adr2001.md) - trb bridge structure  

## Tx

### AddEvidence
Add microreport evidence to an existing dispute. If any of the evidence reports are an aggregate, flag them.  
- `./layerd tx dispute add-evidence [dispute-id] [reports]`

### AddFeeToDispute
Contribute to the first round payment of a proposed dispute if the proposer did not pay the full fee.  
- `./layerd tx dispute add-fee-to-dispute [dispute-id] [amount] [pay-from-bond]`

### ClaimReward
Claim voting rewards after a dispute is resolved. 2.5% of the total dispute fee is rewarded to voters.
- `./layerd tx dispute claim-reward [dispute-id]`

### ProposeDispute
Propose a dispute on a given microreport.  
- `./layerd tx dispute propose-dispute [disputed-reporter] [report-meta-id] [report-query-id] [dispute-category] [fee] [pay-from-bond]`

### Vote
Vote on a given dispute. 33% of power is given to users (tippers), 33% is given to reporters, and 33% is given to the team address.  
- `./layerd tx dispute vote [id] [vote]`

### WithdrawFeeRefund
Allows whoever paid the first round dispute fee to get refunded the fee if the dispute does not get fully funded, or resolves to invalid or support. If the dispute resolves to support, a first round fee payer also gets the disputed reporter's slashed tokens.  
- `./layerd tx dispute withdraw-fee-refund [payer-address] [id]`

### UpdateTeam
Update the team address through governance.
- `./layerd tx dispute update-team [team-address]`

## Queries

### Params
- `./layerd query dispute params`

### TeamVote
- `./layerd query dispute team-vote [dispute-id]`

### Tally
- `./layerd query dispute tally [dispute-id]`

### Disputes
- `./layerd query dispute disputes`

### OpenDisputes
- `./layerd query dispute open-disputes`

### TeamAddress
- `./layerd query dispute team-address`

## BeginBlocker
### CheckOpenDisputesForExpiration
Checks for expired prevote disputes and sets them to failed if expired. Also checks whether any open disputes' vote periods have ended and tallies the vote if so.

### CheckClosedDisputesForExecution
Checks if any disputes are pending execution, and if so, executes the vote.

## EndBlocker
### SetBlockInfo
Checks if a dispute has been opened at the current block height and sets the block info if so.


## Mocks

`make mock-gen-dispute`

## Events
| Event | Handler Function |
|-------|-----------------|
| new_dispute | SetNewDispute |
| added_dispute_round | AddDisputeRound |
| dispute_executed | ExecuteVote |
| fee_added_to_dispute | AddFeeToDispute |
| voted_on_dispute | Vote |


