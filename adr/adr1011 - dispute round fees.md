# ADR 1011: Dispute round fees

## Authors

@akrem

## Changelog

- 2026-06-10: initial version

## Context

A dispute can go through more than one round. Round 1 is the main dispute. If a round does not reach quorum someone can push it to another round by paying a fee, and that fee doubles every round. The chain caps this at 5 rounds.

This ADR is about what the fee is for each round and what happens to it.

Round 1's fee is refundable. You pay the dispute fee, which is the same as the slash amount. If the dispute resolves support you get your fee back minus a small burn and you also get the reporter's slashed tokens. If it resolves invalid you just get your fee back minus the burn. If it resolves against you the reporter gets your fee. So the round 1 fee comes back to you unless you lose the dispute.

Rounds 2 and up are different. The fee you pay to start another round is not refundable to anyone. It is a burn. The whole fee gets consumed, half of it is actually burned and half goes to the voters of the dispute (if no users or reporters voted in any round, the voter half is burned as well). The only thing that is ever refundable is the round 1 fee.

The fee for each extra round starts at 5% of the slash amount and doubles each round. So round 2 is 10% of the slash, round 3 is 20%, round 4 is 40%, round 5 is 80%, capped at 100%. It gets expensive fast on purpose.

Why it works this way:

If a dispute is extended to multiple rounds, the escalating fee does two things:

1. The doubling fee keeps anyone from dragging a dispute out forever. Every extra round costs a lot more than the last one.
2. Half of the round fee goes to the voters to incentivize them to vote on the extra rounds. The fee grows as the rounds go up, so the reward for voting grows with it. Burning all of it instead would just destroy the tokens.

The round 1 fee stays separate from all of this. It is the part that is actually at risk between the disputer and the reporter, and the final round decides what happens to it. The extra round fees are just the cost of asking for another vote.


## Issues / Notes on Implementation

The fee schedule for reference, with s as the slash amount (5% is s/20):

- Round 1: pay s. This is the refundable fee. 5% is consumed (half burned, half reserved as the voter reward; all of it burned if no users or reporters voted), 95% is refundable.
- Round 2: pay s/10 (10% of s). Fully consumed.
- Round 3: pay s/5 (20%). Fully consumed.
- Round 4: pay 2s/5 (40%). Fully consumed.
- Round 5: pay 4s/5 (80%). Fully consumed.

Disputes are capped at 5 rounds.
