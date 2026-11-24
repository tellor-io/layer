# ADR 1011: Dispute round fees

## Authors

@akrem

## Changelog

- 2026-06-10: initial version
- 2026-11-24: retained fee, payer settlement and appeal pricing

## Round one

The opening quote `P` is the slash calculated from the report and dispute category. Payers can contribute cash or bonded stake until the quote is fully funded. A bonded contribution debits exactly the accepted amount. Each position contributes at most its individually floored value. Collection fails atomically if the split cannot be collected at live exchange rates. A first-round receipt lists positive origins from the latest debit first, then earlier origins. Matching delegator and validator pairs share one entry. There is no fixed limit on distinct pairs.

Escrow records the reporter principal actually recovered, `C`. If a positive quote recovers nothing, the transaction fails. If escrow recovers less than the quote, the retained fee `F` becomes the amount recovered. Otherwise the fee stays at the quote. Funding records the retained fee and its burn allocation, `b = floor(F / 20)`. The opening quote, gross contributions and bonded receipts stay on record. Until claimed, bonded excess is outside the owner's stake. It earns no staking rewards and other disputes cannot escrow it.

## Settlement

Anyone can call `WithdrawFeeRefund` for a round-one payer. After an appeal, use the final executed round ID. The calculation uses the payer's gross contribution `g` and the round-one quote. Each leg rounds down independently to whole loya. The module accumulates fractions in millionths of a loya and burns whole loya as they accrue.

| Outcome | Fee returned to its payment source | Reporter principal |
| --- | --- | --- |
| Failed before voting | Gross contribution, limited to the available receipt for bond | None |
| INVALID | `floor(g × (P − b) / P)` | Returned to the disputed reporter's origins |
| SUPPORT | `floor(g × (P − b) / P)` | `floor(g × C / P)` returned through staking for the named payer |
| AGAINST | `floor(g × (P − F) / P)` when `F < P`. When `F = P`, the query shows zero and withdrawal returns an error | The reporter's origins receive `C + F − b` |

The corresponding no-quorum majority follows the same rule. An empty round pays zero on INVALID and SUPPORT. A successful claim consumes the payer row once, including a share that rounds to zero. The query `ClaimableDisputeRewards` shows both legs together. Historical bonded fee returns account for the amount actually collected. Payer-keyed receipts take precedence over the historical hash-only lookup.

For a 100-loya quote and 60 loya recovered, the fee is 60 and the round-one allocation is 3. On AGAINST the reporter receives 117. Payers who contributed 70 and 30 claim 28 and 12 through their respective payment sources. On INVALID and SUPPORT their fee legs are 67 and 29, and their fractions together burn one loya. SUPPORT also allocates 42 and 18 from the reporter's principal.

## Appeals and voters

An unresolved dispute can be appealed up to round five. The offer minimum for opening a dispute applies only to a new dispute. Round `r` from 2 through 5 costs `max(1, min(F, floor(F × 2^(r−1) / 20)))` loya. These percentages are 10%, 20%, 40% and 80%. Even a zero-fee dispute costs one loya to appeal. Each appeal adds its price to `FeeTotal` and `BurnAmount`. The rising cost discourages extending a dispute. The voter reserve rewards participation across rounds.

Upon execution, the allocation consists of the round-one burn allocation plus every appeal fee. If users or reporters voted in any round, reserve half the allocation rounded down for those voters and burn the rest. With no eligible votes, burn the full allocation.

## Coordinated upgrade

Ship #1101 and this settlement change together. The upgrade applies the rules to funding, appeals, execution and claims processed from activation onward. A dispute partly funded before activation uses the new resize when funding completes. Use the stored amounts for earlier rounds, including rounds where #1101 already returned excess. Preserve voter reserves recorded before activation.

For each target chain, use the last committed height before activation (`H − 1`) for the following checks.

1. Identify the running binary and establish whether #1101's funding-time refunds ever ran. Record the chain ID, height, block hash and binary revision with the inventory.
2. Group disputes by first-round ID and follow every appeal to its final round. Include open and partially funded Prevote rounds, rounds awaiting execution, and failed or executed rounds with unpaid payer or voter claims. Distinguish each vote outcome and its no-quorum form. Check the voter records from every round.
3. Reconcile cash payments, bonded receipts and their origins, remaining reporter principal, voter reserves and dust against their backing in the dispute module. Use the amounts still owed after earlier withdrawals. Identify receipts keyed by the dispute hash alone. Several payers may share one such receipt. A successful bonded claim can consume it, and an INVALID or SUPPORT claim can include backing attributed to another payer. Reconcile these rights individually. If #1101 ran, also verify the resulting delegations, validator shares, unbonding entries and staking pool balances.
4. Activate prospectively without a store migration only when the affected rights and staking records are clean or reconciled. Hold activation on an unexplained difference.

### Performance evidence before release

Check mainnet and Palmito separately. On private copies with representative holdings, submit signed cash and multi-owner bonded claims under a finite gas budget justified by observed transactions and node and block limits. Include the two SUPPORT legs, resized AGAINST excess and a retry after cap overflow encounters a full unbonding queue. Record gas limits, gas used, outcomes and changes in balances.

Compare the deployed release and the candidate on equivalent state, hardware, validator sets and consensus settings. Run repeated matched workloads using frequencies from a documented height range, including funding, claims and BeginBlock execution. Report deliberately exercised rare paths separately. For each network, record the builds, sample size, repeat variability and candidate-minus-baseline mean committed block interval. The permitted increase is at most **0.5 seconds on each network**. Record missing measurements explicitly. A keeper benchmark measures execution cost only.
