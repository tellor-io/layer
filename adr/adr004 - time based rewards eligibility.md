# ADR 004: Time based rewards eligibility

## Authors

@themandalore

## Changelog

- 2024-02-22: initial version

## Context

Currently time based rewards (inflationary rewards for reporters) go only to cycle list queries. Ideally we support any query, basically just adding a small amount of inflationary rewards to each piece of data.  The rationale is two fold:
a) subsidizes users needing to tip
b) provides a heart beat for the system in absence of tips (reporters are then around ready to report and we can all see they are reporting accurately)


The issue in just distributing inflationary rewards to all reported data is that there becomes an incentive to report more (unneeded) data in order to increase the amount of rewards given to your reporter.  For instance if you have 10 reporters (equal weight) and they all report for BTC/USD, then they would split the inflationary rewards (if they have unequal weight it would be distributed based upon reporting weight).  The problem is what happens when one of those parties reports for TRB/BTC (imagine a query that only they support)?  For calculation purposes, let's say they report for 9 new queries that only they support.  If the inflation is split based on total reported queries, they had 9 reports and all other reporters (equal weight) also had 9, so our attacker would get 50% of the rewards.   

In order to prevent this we only give inflationary rewards to cycle list queries ( queries that have been voted on by governance that everyone should support at a base level).  


## Alternative Approaches

### tbr directly tied to tips

We could solve this by having it be directly tied to tips.  For this solution, you double tips with inflationary rewards.  This would work like if a tip comes in for BTC/USD for 2 TRB, you add 2 TRB as rewards.  The problem again here is that it incentivizes tipping (an extra step) for a query no one wants (reporters just tip to get the reward) and is hard to police that parties aren't just tipping lowly supported queries to boost their own rewards.  

### only inflation for validators

An easy solution is to keep the inflation for validators and not reporters.  This could be an option, but would not incentivize parties to keep the chain active unless tipped, something we had hoped to do with inflationary rewards. 

### only cycle rewards if consensus reached 

A discussed option was to only provide inflationary rewards to queries that hit consnensus.  This sort of solves the problem, but there would still be the issue that some parties would want to support queries that had 66% support vs those with 100% support.  There would also be a race to submit for more things, (e.g. if I do more than everyone else I'll get more), which is both good, but also fills up the chain unnecessarilty.  

### weight only used once

You could solve the problem by only counting the weight once.  The problem here is that you may be more likely to submit for things that no one can dispute (e.g. we'll just do EVM queries or a static answer like (who is the president?)).  This would take risk away but would not help the chain in anyway.  


## Issues / Notes on Implementation

