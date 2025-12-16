# ADR 1003: Time based rewards eligibility

## Authors

@themandalore
@brendaloya

## Changelog

- 2024-02-22: initial version
- 2024-04-02: clarity
- 2024-04-05: clarity/spelling
- 2024-08-03: clarity
- 2024-12-06: bridge deposits
- 2025-12-09: liveness-weighted distribution

## Context

Currently time based rewards (time based inflationary rewards for reporters) go only to the cycle list queries. Any query can be voted into the cycle list.  The rationale is twofold:

a) subsidizes users' tips

b) provides a heartbeat for the system in the absence of tips (reporters are then ready to report and we can all see they are reporting accurately)

The issue in just distributing inflationary rewards to all reported data is that there becomes an incentive to report more (unneeded) data in order to increase the amount of rewards given to your reporter.  For instance, if you have 10 reporters (equal weight) and they all report for BTC/USD, then they would split the inflationary rewards (if they have unequal weight it would be distributed based upon reporting weight).  The problem is what happens when one of those parties reports for a query that only they support.  For calculation purposes, let's say they don't just do it for one, but report for 9 new queries that only they support.  If the inflation is split based on total reported queries, they had 9 reports(all ones they only support) and all other reporters (equal weight) also had 9 (just for BTC/USD).  In this scenario, if you split the time based reward by weight given, the attacker would get 50% of the rewards. In order to prevent this, we only give inflationary rewards to cycle list queries (queries that have been voted on by governance that everyone should support at a base level).  

Note that bridge deposits are also part of the cycle list (reporters report deposits on the Ethereum bridge contract)

 ![ADR1003: rewards](./graphics/adr1003.png)

## Alternative Approaches

### Time based inflationary rewards(tbr) directly tied to tips

We explored having time based inflationary rewards tied to tips, matching tips with inflationary rewards.  For example, if a tip comes in for BTC/USD for 2 TRB, 2 TRB would be added as rewards.  The problem here is that it incentivizes tipping (an extra step) for a query no one may be using (reporters would tip to get the extra rewards) and it is hard to police that parties aren't just tipping lowly supported queries to boost their own rewards.  

### Only providing time based inflationary rewards for validators

Another option was to keep the inflationary rewards for validators only and not reporters.  However, this would not incentivize parties to keep the chain active unless tipped, something we are aiming to do with inflationary rewards. 

### Only provide rewards if consensus reached 

A discussed option was to only provide inflationary rewards to queries that hit consensus.  This solves the problem of reporters tipping queries only they support, but there would still be the issue that some parties would want to support queries that had 66% support vs those with 100% support to attempt to get more rewards. 

### Weight only counted once

Lastly, another option was to only count the reporter weight once.  The problem here is that you may be more likely to submit for things that no one can dispute (e.g. we'll just do EVM queries or a static answer like (who is the president?)).  This would take risk away but would not help the chain in anyway.  

## Issues / Notes on Implementation

### Liveness-Weighted Distribution (Updated 2025-12)

To further incentivize consistent reporting across ALL cyclelist queries, TBR distribution now incorporates a liveness component. Instead of distributing TBR immediately per aggregate, rewards are accumulated over a configurable period and distributed based on both reporting power AND liveness.

#### Formula

```
liveness = weighted_queries_reported / total_cyclelist_queries
weight = accumulated_power × liveness
reward = (weight / total_weight) × period_TBR
```

Where:
- `weighted_queries_reported` = sum of (1 / opportunities) for each unique query reported
- `total_cyclelist_queries` = number of queries in the cyclelist (e.g., 12)
- `accumulated_power` = sum of reporter's power across all reports in the period

#### Split-Weight for Multiple Opportunities

When a cyclelist query is tipped out-of-turn (while not in rotation), it creates an extra reporting opportunity. To prevent gaming, the weight is split:

- If Q5 appears once (normal rotation): weight = 1.0
- If Q5 appears twice (rotation + out-of-turn tip): each opportunity = 0.5 weight
- Total weight always equals number of cyclelist queries (e.g., 12)

This ensures:
1. Reporters are incentivized to report on ALL opportunities (including out-of-turn tips)
2. Missing an out-of-turn tip has a smaller penalty as far as getting full TBR (0.5 points vs 1.0)
3. No one can dilute others' liveness by tipping many queries

#### Example

With 12 cyclelist queries and Q5 tipped out-of-turn (13 opportunities total):
- Reporter A reports on all 13 → liveness = 12/12 = 100%
- Reporter B reports on 12 (misses out-of-turn Q5) → liveness = 11.5/12 = 95.8%
- Reporter C reports on 11 (misses Q5 entirely) → liveness = 11/12 = 91.7%

#### Configuration

The distribution period is configurable via the `LivenessCycles` governance parameter:
- Default: 1 (distribute every cycle)
- Higher values aggregate over multiple cycles before distribution

#### Key Changes from Previous Implementation

| Aspect | Before | After |
|--------|--------|-------|
| Distribution timing | Per aggregate (immediate) | Per period (batched) |
| Reward basis | Power only | Power × Liveness |
| Out-of-turn tips | Not tracked | Tracked with split-weight |
| Incentive | Report when convenient | Report on EVERY query |

