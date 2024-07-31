# ADR 1009: Handling of Reporter Selections(delegations) and Selectors

## Authors

@themandalore @brendaloya

## Changelog

- 2024-06-25: initial version
- 2024-07-30: Clarifications added 

## Context

In layer, people can delegate their stakes to validators. To distinguish between delegation to validators and delegations to reporters, delegations to reporters are called selections and those making those selections, selectors.

Part of the layer design is that when a reporter submits a price, the system loops through its "selectors" (all parties who have delegated (or selected to give) reporting duties to this reporter) and checks whether or not they are bonded.  The reason for the bonding check is that only "bonded" tokens, ones delegated to one of the top 100 validators, are considered valid for reporting.  One negative of this approach is that the list of selectors could become so large that looping through this list would cause the transactions to fail.  To prevent this, we limited the number of selectors to 100 per reporter. 

However, the cap on selectors and how we handle re-selections can lead to some UX and attack vectors, namely:

- If we allowed instant re-selections you can re-select to another reporter to increase your vote power. To carry out the attack the first reporter would reports early in the report time frame(the user defines the report collection time frame for each queryID), and the second selected reporter goes later.  This could be done multiple times to get extra rewards or manipulate the median. 

- Additionaly, since all reporter tokens have to be bonded (being part of the top 100 validators), there is a scenario where your selected reporter falls out of being bonded.  This means you would effectively lose out on reporting rewards until you reselect.  

- Another issue is that a party can spam the 100 selector limit with tiny amounts to effectively censor the reporter from getting additional selectors and force them to spin up new addresses and pay more gas for reporting multiple times.  

The current solution is to:

- To ensure that any reporter only reports once during the time frame window (which has a maximum of 21 days) and can't reselect to exploit this, re-selection will only be allowed after a lock period of 21 days (meaning that tokens are not counted in any reporters total for that time period).    

- The reporter can set a minimum stake amount that they allow to be selected with.  This prevents cheap spam attacks for larger reporters.

## Alternative Approaches

### Can never re-select

- This would work, but it makes for poor UX for selectors.  In this case if your reporter goes down or you want to switch to yourself, you must unstake both your reporter AND validator, which will lead to a loss in rewards.  With the current method, selectors will only lose out on reporting rewards and the validator delegation can remain untouched.

### Selectors are just locked at first 100 with no minimum

- This was the base approach for just capping.  The issue here is that you have censoring by simply selecting a reporter with a tiny amount in a bunch of addresses in order to fill them up to 100.  The reporter can always start a new address, but this increases their costs as they now must submit twice to report for a given query.  Another option for them would be to get all of their honest selectors to move to a new address, but this coordination is not ideal as it would impose an extra burden on honest actors and require off-chain communication, thus potentially doxing certain parties.  

### Reporters can kick out selectors

- The obvious issue here is censorship (e.g. no selectors from the US).  The goal is to make Tellor as permisionless as possible, so giving the ability to self censor any aspect of the system is best to avoid if possible. One potential solution was to only let them remove if full, but this too can be censored and attacked by the reporter themselves if they wish to censor.

### The reporter gets the top 100 selectors by token weight and the bottom are kicked out

- The problem here is that there is an attack vector of selecting a reporter with a large amount, kicking out all of his selectors and then unselecting that reporter yourself, thus leaving him with no selectors. This  additionaly adds monitoring costs for honest selectors who have to monitor whether they are kicked out of a set.

## Issues / Notes on Implementation

### Cost to attack / prevention method

If censoring is still an issue (filling up the 100 slots), the reporter can always get other redelegators to move and then submit a bad value, thus slashing the attackers tokens.  Therefore the cost of spam is 100 slots * min stake amount, and you'll likely lose it.  We see this being a valid solution as the only reason to attack is to prevent repelators from choosing a specific reporter to increase rewards for your own reporter for a specific period of time.  If the rewards for any time period are much greater than the attack to censor (very likely not the case with a non-zero minimum), censoring could happen for a short period of time.  For this reason, we expect reporters, especially validator/reporters with high reputation, to have larger minimums for selection.  
