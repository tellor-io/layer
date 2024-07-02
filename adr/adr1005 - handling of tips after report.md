# ADR 1005: Handling of tips after report

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-21: initial version
- 2024-03-07: changed to escrow account vs stake
- 2024-04-01: made the distinction clearer for tips not going into the reporter stake
- 2024-04-02: clarity / wording changes
- 2024-06-12: clarify tips locked state

## Context

In layer, when a user sends free floating tokens as a tip, the party(ies) that report for that queryId in the reporting window are given the tip.  Once reported for, these tips are locked in an escrow account that can be withdrawn but observing the same waiting period as unstaking. The tokens are in a locked state until the user calls `WithdrawTip` which delegates the tokens to the specified validator, thereafter the user can undelegate and enter the unbonding period before the tokens are able to be released as free floating tokens.

One of the goals of the Tellor governance structure is to prevent people from farming voting power.  If tips were not locked, a party could tip a random query that only they know how to report for.  They could report, then tip those exact same tokens, thus looping them in order to look like there is ton of tip demand for this query and increase their 'user' and 'reporter' voting power.  The two methods to prevent this are a) a 2% fee on tips and b) locking the tip in escrow.  Now if the party wishes to withdraw that tip, they will have to wait the 21 day unbonding period, similar to the unstaking.  

Another potential attack is that they could use tipping to bypass the deposit stake caps.  Notably, they could tip a large portion of the entire staked supply (e.g. 66%) and then halt the chain or force a fork.  

## Alternative Approaches

### Higher fee burn on tips

The other method we could do is just increase that 2% fee to something like 5 or 10%, thus completely draining the attacker if they did this method.  Although this could be a solution, limiting that fee is important because it supports having on-chain tips (can be tracked for governance/ usage purposes) vs off-chain.  It also only mildly fixes the depositStake bypass.  

### Tips locked as stake and limit the amount

We could just lock the tips as stake, but then tipping a query only you report is essentially a "deposit stake" function.  This could be handled by just limiting the amount you could unlock, but this dual structure could cause issues for tracking voting power and stake.  Keeping it in a separate escrow was deemed cleaner in the code.

### just cap tip size or total amount

One could just cap the tip size, but this could easily be avoided by just splitting up tips.  We had also discussed limiting total tips to some % of total stake (e.g. 2%), but then a) you could get around the deposits by just tipping smaller amounts for fast queries; and b)  you could potentially censor the system by capping out the amount of tips (i.e. if you tip to the limit, now no one can do a legitimate tip).

## Issues / Notes on Implementation
