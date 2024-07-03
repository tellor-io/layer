# ADR 1009: Handling of Reporter Delegations - Selectors

## Authors

@themandalore

## Changelog

- 2024-06-25: initial version

## Context

Part of the layer design is that after a reporter submits a price, the system loops through her "selectors" (all parties who have delegated (or selected to give) reporting duties to this reporter) and checks whether or not they are bonded.  The reason for the bonding check is that only "bonded" tokens or ones delegated to one of the top 100 validators are considered valid for reporting (we need to keep the token amounts for each set equal).  One negative of this approach is that the list of selectors could become so large that looping through this list would cause the transation to fail.  To prevent this, we limited the number of selectors to 100 per reporter.   This makes sense but there are some UX and attack vectors that arise from this, namely:

- Do you allow re-selections, ie selecting a different reporter?  If yes, how fast?  If you allow instant re-selections you can re-select to another reporter to increase your vote power (the first reporter reports early in the report time frame, and the second selected reporter goes later.  You could even do this multiple times) to get extra rewards or manipulate the median.  This is a non-starter.  Additionaly, since we check for bonding status (part of the top 100 validators), there is a scenario where your selected reporter falls out of being bonded.  This means you would effectively lose out on reporting rewards until you reselect.  

- There are additional issues where a party can spam the 100 selector limit with tiny amounts to effectively censor the reporter from getting additional selectors and force them to spin up new addresses and pay more gas for reporting multiple times.  

The current solution is to:

- Allow for re-selection after a lock period where the tokens are not counted in any reporters total for that time period.  The time will be the maximum report time frame of 21 days.  

- The reporter can set a minimum stake amount that they allow to be selected with.  This prevents cheap spam attacks for larger reporters.

## Alternative Approaches

### Can never re-select

- This would work, but it makes the UX for selectors very bad.  In this case if your reporter goes down or you want to switch to yourself, you must unstake both your reporter AND validator...which will lead to a loss in rewards.  With the current method, selectors will only lose out on reporting rewards and the validator delegation can remain untouched.

### Selectors are just locked at first 100 with no minimum

- This was the base approach for just capping.  The issue here is that you have censoring by simply selecting a reporter with a tiny amount in a bunch of addresses in order to fill them up to 100.  The reporter can always start a new address, but this increases their costs as they now must submit twice to report for a given query.  Another option for them would be to get all of their honest selectors to move to a new address, but this coordination is not ideal as it would impose an extra burden on honest actors and require off-chain communication, thus potentially doxing certain parties.  

### Reporters can kick out selectors

- The obvious issue here is censorship (e.g. no selectors from the US).  The goal is to make Tellor as permisionless as possible, so giving the ability to self censor any aspect of the system is best to avoid if possible. One potential solution was to only let them remove if full, but this too can be censored and attacked by the reporter themselves if they wish to censor.

### The reporter gets the top 100 selectors by token weight and the bottom are kicked out

- The problem here is that there is an attack vector of selecting a reporter with a large amount, kicking out all of his selectors and then unselecting that reporter yourself, thus leaving him with no selectors. This  additionaly adds monitoring costs for honest selectors who have to monitor whether they are kicked out of a set.

## Issues / Notes on Implementation

### Cost to attack / prevention method

If censoring is still an issue (filling up the 100 slots), the reporter can always get other repelegators to move and then submit a bad value, thus slashing the attackers tokens.  Therefore the cost of spam is 100 slots * min stake amount, and you'll likely lose it.  We see this being a valid solution as the only reason to attack is to prevent repelators from choosing a specific reporter to increase rewards for your own reporter for a specific period of time.  If the rewards for any time period are much greater than the attack to censor (very likely not the case with a non-zero minimum), censoring could happen for a short period of time.  For this reason, we expect reporters, especially validator/reporters with high reputation, to have larger minimums for selection.  
