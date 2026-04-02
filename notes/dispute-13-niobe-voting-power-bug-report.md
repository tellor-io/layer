# Dispute 13 Reporter Voting Power Bug

## Summary

Reporter `tellor16vuuwx7lekkfy57nl9mxmxrplfzflgylxzg987` ("niobe") could not vote on dispute `13` even though it was an active reporter with current stake and recent aggregates. The chain rejected the vote with `voter power is zero`.

The likely root cause is that dispute voting reads from the historical `reporter.Report` snapshot store, while active reporting can continue without rewriting that snapshot after pruning if the delegation hash is unchanged.

## Chain Evidence

- Dispute `13` is in voting and uses `block_number = 16401121`.
- Vote tx `FB6E6ADF09B7C5A3B7D35E85C558FE80A13019B2A0A07AC16C0900CACF70313E` failed with:
  - `raw_log: failed to execute message; message index: 0: voter power is zero`
- Current reporter state for `tellor16vuuwx7lekkfy57nl9mxmxrplfzflgylxzg987`:
  - reporter exists
  - `jailed = false`
  - current reporter `power = 1417`
- Current selector set for the reporter:
  - self selector `tellor16vuuwx7lekkfy57nl9mxmxrplfzflgylxzg987`: `45172124`
  - selector `tellor1edr39pfjd2j0l7335dx2zkgk5zmtl5cwnuh6xw`: `1372505522`
- Reporter lifecycle:
  - `MsgCreateReporter` at height `4080`, timestamp `2025-05-09T16:22:18Z`
  - selector `tellor1edr39pfjd2j0l7335dx2zkgk5zmtl5cwnuh6xw` selected niobe at height `8135`, timestamp `2025-05-09T18:08:08Z`
  - no `MsgSwitchReporter` found for either niobe or `tellor1edr...`
- Reporter activity just before the dispute:
  - `get_aggregate_before_by_reporter(...)` returned an aggregate by niobe at height `16401099`
  - dispute block is `16401121`
  - reporter was therefore active only 22 blocks before the dispute snapshot block

## Code Path

Dispute voting uses reporter power at the dispute's stored block and errors if the resulting total power is zero.

- `x/dispute/keeper/msg_server_vote.go`
- `x/dispute/keeper/vote.go`
- `x/reporter/keeper/keeper.go`

Relevant behavior:

1. `MsgVote` calls `SetVoterReporterStake(..., dispute.BlockNumber, ...)`.
2. `SetVoterReporterStake()` calls `reporterKeeper.GetReporterTokensAtBlock()`.
3. `GetReporterTokensAtBlock()` reads from the historical `Report` indexed map via `GetDelegationsAmount()`.
4. If the `Report` lookup returns a nil total, reporter voting power is treated as zero.

## Root Cause

`ReporterStake()` only persisted a new `Report` snapshot when `handlePeriodTracking()` reported that the delegation hash changed.

That interacts badly with `PruneOldReports()`:

- `PruneOldReports()` deletes old `Report` snapshots after the retention window.
- If a reporter's selector set remains unchanged for long enough, all old `Report` snapshots can be pruned away.
- The reporter can still keep reporting successfully because the live stake calculation still works.
- But if the delegation hash is unchanged, `ReporterStake()` would not rewrite a fresh `Report` snapshot.
- A later dispute vote then looks up historical reporter power, finds no surviving snapshot, and gets zero voting power.

In short:

- current live reporter power exists
- historical dispute snapshot is missing
- dispute vote fails with `voter power is zero`

## Fix Applied

Patched `x/reporter/keeper/reporter.go` so that `ReporterStake()` rewrites a `Report` snapshot whenever no historical snapshot survives for that reporter, even if the delegation hash is unchanged.

This preserves existing period-tracking behavior while restoring the dispute snapshot that voting depends on.

## Regression Test Added

Added a unit test in `x/reporter/keeper/reporter_test.go` that simulates:

- a reporter with valid stake
- an existing `ReporterPeriodData` entry whose hash matches current stake
- no surviving `Report` snapshot

Expected result:

- `ReporterStake()` recreates the missing `Report` snapshot
- `GetReporterTokensAtBlock()` returns the expected nonzero power
