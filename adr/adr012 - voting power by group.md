# ADR 012: voting power by group

## Authors

@themandalore

## Changelog

- 2024-02-27: initial version
- 2024-04-01: 

## Context

For voting on disputes, who gets to say what a good value is?  The current split:

25% reporters
25% users (based on tips)
25% team
25% token holders

Most notable in the split is the absence of validators and relayers.  The rationale for not including validators is the treatment of delegated tokens.  Currently delegated tokens are already counted twice (token holders and reporters they are delegated to).  If the validator was also able to use tokens delegated to themselves, you could essentially delegate tokens to yourself as a reporter and a validator and then triple your voting power.  We could choose between giving the power to reporters or validators in this case and we went with reporters for the same reason disputes can be started with delegated tokens from reporters and not validators; data reporting and quality is done by the reporters, chain operations are done by the validators. 



## Alternative Approaches

### add in validators

Adding in validators (or having validators rather than reporters) could introduce a new stake holder set that could help to decentralize the voting set futher.  Additionally, it might be said that reporters are biased due to their obvious conflict of interest and that disputes should be resolved by neutral third parties.  

This is valid, however the attack method of tripling tokens by dual self degation could undermine chain security.  We feel that the current split (reporters over validators) gives them a say in the accuracy of the vote and gives weight to their long term interest in the validity of the data.  It is also an unknown how different validator and reporter sets will be.  There is a substantial chance that they will overlap significantly and the two sets will not require double counting. 

### remove team

A long term plan isto further decentralize the protocol by removing the team's voting weight on disputes. In the short term, as the protocol matures to team acts as a tie breaker.

### different reporter voting weight calculation 

Instead of just using reporter weight as the percentage of total reporter stake, we could use a counting methodology similar to current tellor (each report counts as one vote, regardless of weight).  This has benefits of supporting smaller reporters as much as larger ones, as well as hardening over time as the voting power is non-transferrable.  

The downside here is that votes are still sellable, you just need to sell your private key.  This actually becomes dangerous as reporters who want to exit are incentivized to sell their voting power to attackers once they are unstaked.  You could fix this by also requiring them to be staked, but it only changes the attack cost, not the fact it exists as an exit strategy.  

This is the case for users as well. Users do not have to stake and they earn voting weight by the amount of tips the provide. However, users are dissinsitivized from selling their private keys because doing so could trigger an attack to their own protocol (there is no guarantee an attacker that is willign to buy their keys will not attack them too). 

Brenda Q--Should we decrease user voting power over time or reduce it if they stop tipping over two months???

### move to different goverance structure

In the long run, it is definitely on the table to move to a different governance system.  Whether it's liquid staking, delegations, or other market forces that could drive centralization and alter crypto economic incentives, tellor remains commited to having an active community with a robust censorship resistant core that comes to consensus on any data.  Whether this looks like a traditional one citizen one vote system, a reputation system, or just further refinement in the split of voting powers, tellor is open to exploring these options as the crypto ecosystem does.  As of now, DAO governance is nascent and fragile.  We're currently governance minimalists that want to push the boundaries in areas other than decentralized governance.  That said, tellor has a social layer that is the ultimate fallback and this alone was a huge reason for becoming a standalone chain.  


## Issues / Notes on Implementation

