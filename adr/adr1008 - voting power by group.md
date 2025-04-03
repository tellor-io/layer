# ADR 1008: Voting power by group

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-27: initial version
- 2024-04-01: clarified language
- 2024-04-02: added option on decreasing user vote power
- 2024-08-03: clean up
- 2025-01-14: remove token holder voting group
- 2025-04-01: clarifications 

## Context

For voting on disputes and determining if a value is correct, these groups are weighted evenly:

- 33.3% reporters (based on reporting power)
- 33.3% users (based on total number of tips)
- 33.3% team

Most notable in the split is the absence of validators, relayers, and token holders.  The rationale for not including validators is the treatment of delegated tokens.  Currently delegated tokens are already counted twice (token holders and reporters they are delegated to).  If the validator was also able to use tokens delegated to themselves, you could essentially delegate tokens to yourself as a reporter and a validator and then triple your voting power.  We had to choose between giving the power to reporters or validators in this case and we chose reporters for the same reason disputes can be started with delegated tokens from reporters and not validators; data reporting and quality is done by the reporters, chain operations are done by the validators. Also, in old tellor, reporters could earn voting weight per report count, however that has been removed in Layer in favor of reporter weight.

In old tellor we also included token holders.  The reason against is that snapshotting token balances can get expensive and voting can be gamed if you don't use a static block number for balances.  The rationale for accepting this is that there will be few free floating tokens on Layer.  As the token lives on Ethereum with most of the balances on CEX's, the idea that people will bridge their tokens and NOT delegate to a reporter should be rare.  Additionally, the counting of token holders was double counting those delegated to reporters.  Now the tokens more accurately balance power between reporters and users. 

How Delegation Works (an example)- If A and B each have 100 tokens and A and B both delegate to B for reporting.  For voting, reporter weights go to B and the token weight portion also goes to B for 200, unless A votes.  If A votes, he gets his 100 and B gets 100 (note that this is standard for validator delegation and votes in the cosmos sdk).

## Alternative Approaches

### Add in token holders

We had originally wanted to give free floating token holders weight, but it is unlikely they will be undelegated.  Additionally, token holders on ETH always have the option to fork the value of the token away if they see a compromise (the ETH token is what's listed on exchanges). 

### Add in validators

Adding in validators as an additional weighted group for settling disputes (or having validators rather than reporters) could introduce a new stakeholder set that could help to decentralize the voting set further.  Additionally, it might be said that reporters are biased due to their obvious conflict of interest and that disputes should be resolved by neutral third parties.  

This is valid argument, however the attack method of tripling tokens by dual self delegation could undermine chain security.  We feel that the current split (reporters over validators) gives them a say in the accuracy of the vote and gives weight to their long term interest in the validity of the data.  It is also an unknown how different validator and reporter sets will be.  There is a substantial chance that they will overlap significantly and the two sets will not require double counting. 

### Remove team

The long term plan is to further decentralize the protocol by removing the team's voting weight on disputes and exploring other governance structures. In the short term, the team acts as a tie breaker.

### Different reporter voting weight calculation 

Instead of using reporter weight as the percentage of total reporter stake, we could use a counting methodology similar to current tellor (each report counts as one vote, regardless of weight).  This has benefits of supporting smaller reporters as much as larger ones, as well as hardening over time as the voting power is non-transferrable.  

The downside here is that votes are still sellable, since reporters could sell their private key.  This actually becomes dangerous as reporters who want to exit are incentivized to sell their voting power to attackers once they are unstaked.  You could fix this by also requiring them to be staked, but it only changes the attack cost, not the fact it exists as an exit strategy.  

This is the case for users as well. Users do not have to stake and they earn voting weight by the amount of tips the provide. However, users are disincentivized from selling their private keys because doing so could trigger an attack to their own protocol (there is no guarantee an attacker that is willing to buy their keys will not attack them too). 

### Decrease user voting power over time

One option is to reduce user voting power over time.  The risk is that a user will be big, gain a lot of power and then want to sell it once they no longer need it.  By reducing their power over time, this could remove this risk and give more credence to currently tipping parties.  

The downside here is that the risk is still alive (the user just has to sell it sooner) and it fails to recognize longevity. The rate of attrition/aging would have to take into account both, length of time and amount of tips. Otherwise, if a user tips faithfully once a day for 2 years could end up with less power than a user who tips 10 times a day for a week. Obviously for voting, we'd prefer the long term users and implementing a proper attrition rate could be tricky. 

Another way to fix the attack of buying vote power via tips is to have others recognize the attack and also tip on subsequent vote rounds, so the system would always be safe if >50% of the power is honest; and as last resort a social fork could be done. 

### Move to different governance structure

In the long run, we are open to move to a different governance system.  Whether it's liquid staking, delegations, or other market forces that could drive centralization and alter crypto economic incentives, tellor remains committed to having an active community with a robust censorship resistant core that comes to consensus on any data.  Whether this looks like a traditional one citizen one vote system, a reputation system, or just further refinement in the split of voting powers, tellor is open to exploring these options as the crypto ecosystem does.  As of now, DAO governance is nascent and fragile.  We're currently governance minimalists that want to push the boundaries in areas other than decentralized governance.  That said, tellor has a social Layer that is the ultimate fallback and this alone was a big part of the reason for becoming a standalone chain.  

## Issues / Notes on Implementation


