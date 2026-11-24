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
- [ADR 1011](../../adr/adr1011%20-%20dispute%20round%20fees.md) covers fee settlement and appeal prices.
- [ADR 1012](../../adr/adr1012%20-%20stake%20concentration%20caps.md) describes headroom and unbonding on stake returns.

## Tx

### AddEvidence
Add microreport evidence to an existing dispute. If any of the evidence reports are an aggregate, flag them.  
- `./layerd tx dispute add-evidence [dispute-id] [reports]`

### AddFeeToDispute
Contribute to the first round payment of a proposed dispute if the proposer did not pay the full fee. Funding the opening quote starts escrow. If escrow recovers a positive amount below the quote, that amount becomes the retained fee. Each payer's gross contribution stays in the dispute module until settlement.

- `./layerd tx dispute add-fee-to-dispute [dispute-id] [amount] [pay-from-bond]`

### ClaimReward
Claim a user or reporter voting reward after execution. When users or reporters voted, their reserve is half the sum of the round-one burn allocation and appeal fees, rounded down.
- `./layerd tx dispute claim-reward [dispute-id]`

### ProposeDispute
Propose a dispute on a microreport or appeal an unresolved round. Appeals cost 10%, 20%, 40% and 80% of the retained fee in rounds two through five respectively, rounded down with a minimum of one loya.

- `./layerd tx dispute propose-dispute [disputed-reporter] [report-meta-id] [report-query-id] [dispute-category] [fee] [pay-from-bond]`

### Vote
Vote on a given dispute. 33% of power is given to users (tippers), 33% is given to reporters, and 33% is given to the team address.  
- `./layerd tx dispute vote [id] [vote]`

### WithdrawFeeRefund
Anyone can settle a first-round payer's claim by naming the payer and the failed or final executed dispute ID. A failed dispute returns the full cash contribution. For a bonded contribution it returns the smaller of the payment and the tracked receipt. INVALID and SUPPORT return the payer's share of the opening quote less the burn on the retained fee. SUPPORT also returns the payer's share of recovered reporter principal through staking. AGAINST returns the payer's share of any excess above the retained fee.

Cash refunds go to the payer. Bonded refunds go through the original delegators' staking positions, with unbonding for cap overflow. If a required unbonding queue is full, retry after an entry matures. A successful withdrawal settles all legs for that payer once.

- `./layerd tx dispute withdraw-fee-refund [payer-address] [id]`

### UpdateTeam
Update the team address through governance.
- `./layerd tx dispute update-team [team-address]`

## Queries

### Params
- `./layerd query dispute params`

### ClaimableDisputeRewards

Query the named address and final dispute ID for the payer entitlement and voter reward. `FeeRefundAmount` includes both SUPPORT legs. A query leaves the claim available even when withdrawing it would require unbonding capacity.

- `./layerd query dispute claimable-dispute-rewards [dispute-id] [address]`

### TeamVote
- `./layerd query dispute team-vote [dispute-id]`

### Tally
- `./layerd query dispute tally [dispute-id]`

### Disputes

`SlashAmount` is the opening quote, `DisputeFee` is the retained round-one fee and `FeeTotal` includes gross payments across rounds. A fully funded opening reports its retained fee in `new_dispute.total_fee`. If a later top-up triggers the resize, read `dispute_fee_adjusted` for the quote, the recovered amount and the burn allocation.

- `./layerd query dispute disputes`

For upgrade operators, ADR 1011 describes the height-pinned inventory of outstanding claims, the backing checks and the mainnet and Palmito block-time criterion. [Coordinated upgrade](../../adr/adr1011%20-%20dispute%20round%20fees.md#coordinated-upgrade)

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
| dispute_fee_adjusted | adjustDisputeFeeToCollectedSlash (opening or final top-up) |
| added_dispute_round | AddDisputeRound |
| dispute_executed | ExecuteVote |
| fee_added_to_dispute | AddFeeToDispute |
| voted_on_dispute | Vote |
| team_address_updated | UpdateTeam |


