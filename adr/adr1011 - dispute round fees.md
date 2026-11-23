# ADR 1011: Dispute round fees

## Authors

@akrem

## Changelog

- 2026-06-10: initial version
- 2026-11-21: receipt-based funding and settlement, priced payer admission, appeal and burn rounding (PR #1101 review)

## Context

A dispute can go through more than one round. Round 1 is the main dispute. If a round does not reach quorum someone can push it to another round by paying a fee. The scheduled percentage doubles with each appeal; the chain caps disputes at 5 rounds.

This ADR is about what the fee is for each round and what happens to it.

Round 1's fee is refundable. You pay the dispute fee, which is the same as the slash amount. If the dispute resolves support you get your fee back minus a small burn and you also get the reporter's slashed tokens. If it resolves invalid you just get your fee back minus the burn. If it resolves against you the reporter gets your fee. So the round 1 fee comes back to you unless you lose the dispute.

Round-one funding credits only coins transferred to the dispute module.
Share conversion can make a bond payment deliver less than requested, so any
difference remains fundable until the quote is covered. When the collectible
slash is lower than the quote, retain only the collected amount and return
each payer's excess in proportion to actual receipts. Assign indivisible
loya by largest remainder, with ties by payer key order. Disputes funded
before receipt-based accounting settle reporter wins on the backed
round-one fee and its allocation, not on the outstanding nominal deficit.

The first 32 distinct payers (including the proposer) may make any positive
partial contribution. After that, a new partial payer must deliver at least
`ceil(quote / 100)` loya. Existing payers may still top up and anyone may
deliver the exact balance to open voting. Before completion, at most 99
additional payers can meet the floor; the final payment adds at most one
more payer. This bounds new round-one payer rows, not the number of staking
origins and not payer rows accumulated before activation.

Rounds 2 and up are different. The fee you pay to start another round is not refundable to anyone. It is added to the amount consumed at settlement. Eligible user and reporter voters receive a reserve of half that total, rounded down to whole loya; the remainder is burned. If there are no eligible voters, the whole allocation is burned. The only fee that is refundable is the round 1 fee.

The fee for each extra round starts at 5% of the slash amount and doubles each round. So round 2 is 10% of the slash, round 3 is 20%, round 4 is 40%, round 5 is 80%, capped at 100%. It gets expensive fast on purpose.

Why it works this way:

If a dispute is extended to multiple rounds, the escalating fee does two things:

1. The doubling fee keeps anyone from dragging a dispute out forever. Every extra round costs a lot more than the last one.
2. Half of the round fee goes to the voters to incentivize them to vote on the extra rounds. The fee grows as the rounds go up, so the reward for voting grows with it. Burning all of it instead would just destroy the tokens.

The round 1 fee stays separate from all of this. It is the part that is actually at risk between the disputer and the reporter, and the final round decides what happens to it. The extra round fees are just the cost of asking for another vote.


## Issues / Notes on Implementation

The fee schedule for reference, with s as the slash amount (5% is s/20):

- Round 1: pay s. The consumed portion is `floor(s/20)`; the remainder is at risk or refundable based on the outcome.
- Round 2: pay s/10 (10% of s). Fully consumed.
- Round 3: pay s/5 (20%). Fully consumed.
- Round 4: pay 2s/5 (40%). Fully consumed.
- Round 5: pay 4s/5 (80%). Fully consumed.

For each appeal, apply the scheduled percentage first, then round down to whole
loya. If `s` is positive, the fee is at least 1 loya and at most `s`. The
one-loya floor means tiny fees do not necessarily double each round. For
example, `s = 15` costs 1, 3, 6, and 12 loya in rounds 2–5. Network transaction
fees are separate. An appeal must deliver its entire fee in one transaction;
if share rounding prevents exact payment from bond, pay from an account balance.

Disputes are capped at 5 rounds.
