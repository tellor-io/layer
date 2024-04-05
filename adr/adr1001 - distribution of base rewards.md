# ADR 1001: Distribution of base rewards

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-27: initial version
- 2024-04-01: expanded on discussion

## Context

Currently base (inflationary) rewards are split between reporters and validators on a 75/25 split.  75 percent of the inflationary reward (~3000 TRB/month) is given to reporters of the cycle list, proportionally to their reporting stake used in each aggregated query.  25 percent of the reward (~1000 TRB/month) is given to the validators as their base reward.  

The split was chosen arbitrarily (and we may change it later) but with the goal of incentivizing a significantly larger number of reporters since the validator set is capped. This split should incentivize validators to also become reporters but does not force them to. Keeping a clear distinction between the roles and not forcing validators to be reporters can allow cosmos validators to 'easily' become validators in Layer as well. 


## Alternative Approaches

### All base reward given to validators

Should you give all rewards to the validators, you can incentivize parties to stake more and validate, something that chains are usually measured on (stake/security).  A large portion of cosmos validators can be expected to participate in vanilla chain validation, however whether they will report (especially on the more manual queries) is yet to be seen.  We want to parse the rewards to incentivize parties to do both, but also have an even broader reporter set for smaller queries (so it does not get prohibitively expensive to tip /report for weird unsupported queries).  

### All rewards given to reporters

This option is attractive because you would essentially make the 100 biggest reporters also the validators.  Since non-reporting validators would have no reason to report, reporters would simply add on the additional duty of validation so they could get their rewards.  

The downside here is that there may be costs to being a validator (hardware, memory, etc.) that may make it a race to the bottom in terms of being a validator.  If no one wants to be a validator other than attackers or reporters for the basic reason of having a chain, the issue is that reporters are in contest to do as much reporting but as little validation as possible.  We'd rather have it such that the costs of validating are covered and it's competitive, but reporters can do it also.  You want competition to be a validator and making sure it's not game theoretic outcome of no validators is obviously important. 

### Adding in relaying

Another duty for the chain is relaying.  Although updating prices on the Tellor chain is important, it's ultimately meaningless if no one can get the data on their own chain.  Relaying is a trustless role, however as we know from current Tellor, covering gas costs to get data on-chain can be important.  One option could be to refund relayers for moving data to user chains.  

The issue here is in the book keeping and governance.  How do we know which chains to refund what amount?  We want to avoid a situation where parties are pushing data unnecessarily.  It could be an option to refund say 90% of gas costs, but this would mean that we are subsidizing more expensive chains more, something we may not want to incentivize.


## Issues / Notes on Implementation

These are parameters.  When more data comes in as to the distribution/ security of the network, we can make a more educated choice on the split. 