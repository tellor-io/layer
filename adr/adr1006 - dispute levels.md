# ADR 1006: Dispute Levels

## Authors

@themandalore

## Changelog

- 2024-02-22: initial version
- 2024-02-22: spelling

## Context

Tellor disputes have three categories:  warning, minor, and major.  The decision for three is even arbitrary.  The point is to separate out major disputes (e.g. attacks) from minor disputes such as being slightly off for a calculation or having a single API failure.  To three categories are as follows:

    * Warning (dispute fee is 1% of stake)- jail, but no lock, can call a function to be released from jail and begin reporting again
    * Minor Infraction (dispute fee is 5% of stake) - jailed for 10 minutes and out when they call the release from jail function
    * Major Infraction (dispute fee is 100% of stake) - jail until dispute over ( since 100% of stake).

A release function has to be called after a warning or minor infraction to ensure the staker has looked at the dispute and implemented a fix as necessary. Infractions in these lower two tiers can generally be assumed to not be malicious. 

After specifying the dispute category, the disputer will submit an amount of TRB up to the minimum slashing amount before the dispute can initiate. If they don’t have enough funds themselves, for up to one day, others can add to the pot until they hit the slashing amount(1, 5, or 100 percent depending on the slashing category).  Once the amount is hit (could be hit instantly upon proposing the dispute, or could take up to a day), the potential slashing amount will be taken from the disputed validator and placed into a locked stake.

To clarify, a warning is more of a pause.  For example, your machine submitted a bad value.  Address it and click "unjail" to resume again.  
A minor infraction will usually come after a warning, for example, you're submitting 1% under everyone else repeatedly and threatening to pull down the median.  Finally a major infraction should be saved for attacks.  The reporter is slashed entirely.  For example, if a bad value is submitted for (e.g. BTC 1M), we can assume it's not an attack and submit a warning.  If however, they unjail themselves and continue to submit bad values, they run the risk of being slashed entirely. 


## Alternative Approaches

### no categories - free floating percentages

One option is to just have the disputer pick a percent of the reporters stake and submit for a dispute.  Then jail time could also be on a scale (up to 2 days (vote time) as a percent of stake).  This would be relatively straight forward to code, but we'd expect categories and norms to pop up similar to what we proposed.  For new reporters and disputers it could also be unclear as to how much to dispute. By having clear categories with examples, we hope to minimize the chance of full disputes being open for minor issues. 

### different weights for each category

1, 5, and 100 we're picked relatively arbitrarily.  The 100% is obviously correct for stopping an attack, but we went back and forth between 1 and 5 vs higher numbers like 5, 10 or even 10, 25.  Ultimately the cosmos ecosystem is much more exposed to smaller slashing penalties and the larger amounts seem unnecessary if we have a sufficiently decentralized reporter set.  Additionally, these levels can be adjusted by governance if they seem insufficient. 

## Issues / Notes on Implementation

One issue to keep in mind with regard to jail times and slashing is that it is likely that LST's pop up for reporter/validator stakes.  Should this happen, reporters will likely have little lock up with regard to posting bad prices and we can opt for faster freezing as its not their stake that they are actually losing.  This is a problem with any delegated system and must be monitored that incentives and slashing conditions properly discentivize operators even if it is not their capital.  


## References / Links

[https://forum.celestia.org/t/defending-against-lst-monopolies/1554/27](https://forum.celestia.org/t/defending-against-lst-monopolies/1554/27)
