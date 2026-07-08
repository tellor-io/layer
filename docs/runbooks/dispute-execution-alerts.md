# Dispute Execution Alerts

## Signals

`CheckClosedDisputesForExecution` processes at most 1000 pending disputes per
block in deterministic dispute-id order. Metrics are bounded-window signals for
that inspected set, not global pending-dispute totals.

- Event `dispute_execution_failed`: emitted when `ExecuteVote` returns an
  unexpected/non-cap error. Alert on any occurrence.
- Event `dispute_execution_deferred`: emitted for known retryable
  stake-distribution errors.
- Counter `dispute.execution.failed`: increments with
  `dispute_execution_failed`. Page when the counter increases above zero in a
  production window.
- Counter `dispute.execution.deferred`: increments with
  `dispute_execution_deferred`. Investigate sustained growth.
- Gauge `dispute.execution.processed_count`: number of pending disputes
  inspected in the bounded window.
- Gauge `dispute.execution.max_deferral_age_seconds`: max age, over the
  bounded window, from `DisputeEndTime` to the current block time. Alert when it
  exceeds the operator-defined dispute execution SLO.

## Response

1. Inspect the event attributes `dispute_id` and `reason`.
2. Query the dispute and confirm `PendingExecution` remains true after the
   failed/deferred block.
3. For `dispute_execution_failed`, treat the reason as a root-cause bug or
   operational fault. The chain should not halt, but the dispute will retry and
   alert each block until fixed.
4. For `dispute_execution_deferred`, first check validator/delegator/reporter
   cap headroom and staking exchange-rate state. Normal cap overflow should now
   complete through unbonding overflow; repeated deferral usually means a
   non-headroom staking admission condition.
5. Do not replace these bounded-window metrics with an in-BeginBlock full
   pending-dispute scan. Use off-chain queries if exact global pending count or
   global max age is required.

## Sources

- `x/dispute/abci.go`: event emission, counters, bounded-window gauges, and
  cache rollback behavior.
- `adr/adr1012 - stake concentration caps.md`: dispute execution policy,
  residual risks, and future `MaxDeferralBlocks` direction.
