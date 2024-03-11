# ADR 002: Handling of tips after report


## Authors

@themandalore

## Changelog

- 2024-02-21: initial version

## Context

In layer, when a user sends free floating tokens as a tip, the party(ies) that report for that queryId next cycle are given the tip.  Once reported for, these tips are locked as reporting stake.  This is because of how tellor governance is structured, namely to prevent people from farming voting power.  

If they were not locked, a party could tip a random query that only they know how to report for.  They could report, then tip those exact same tokens, thus looping them in order to look like there is ton of tip demand for this query.  The two methods to prevent this are a) a 2% fee on tips and b) locking the tip as a reporting stake.  Now if the party wishes to withdraw that tip, they will have to wait the 21 day unbonding period.  


## Alternative Approaches

### Higher fee burn on tips

The other method we could do is just increase that 2% fee to something like 5 or 10%, thus completely draining the attacker if they did this method.  Although this could be a solution, limiting that fee is important because it supports having on-chain tips (can be tracked for governance/ usage purposes) vs off-chain.  


## Issues / Notes on Implementation

