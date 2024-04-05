# ADR 1004: Fees on Tips

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-21: initial version
- 2024-04-01: added to discussion
- 2024-04-02: expanded upon alternative options

## Context

A fee on tips is necessary to discourage vote farming and spamming the network by tipping data that has no support. 

The 2% fee on tips was chosen relatively arbitrarily, but is consistent with the 2% fee that has been present on previous versions of [Tellor's autopay contract](https://github.com/tellor-io/autoPay). 


## Alternative Approaches

### No Fee

Rather than a fee for profit, the fee in the tellor system acts more as a spam prevention tool. Removing the fee is unfortunately not an option as it can encourage vote farming. In Layer, users who tip receive vote power based upon how much they tip.  An attack vector is present where someone could just tip a query only they support, report for it, and gain voting power without losing funds.  The fee prevents this (or makes it costly).  In old tellor, reporters could earn voting weight per report count (and double the effectiveness of this attack), however that has been removed in Layer in favor of reporter weight. 

### Higher fee

We could just increase that 2% fee to something like 5 or 10%.  Although this could be a solution, limiting that fee is important because it supports having on-chain tips (can be tracked for governance/ usage purposes) vs off-chain.  The big discussion to continue to analyze is balancing the need for a fee (to prevent vote farming attacks) and the risk of the fee being too high that only off-chain tips are used and we lose the ability to give actual users voting power (a say in the validity of the data).   


## Issues / Notes on Implementation

The fee can be increased if vote farming or spamming is observed. 