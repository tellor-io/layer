# ADR 1012: Reporter power cap

## Authors

@danflo27

## Changelog

- 2026-06-12: initial version
- 2026-06-12: disable interchain accounts (host and controller) in the same upgrade after finding mainnet's ICA host allowlist set to `["*"]`; only interchain queries remain supported
- 2026-06-12: documented the decision to leave the delegator 30% cap hardcoded while the reporter cap is a param

## Context

Layer already limits stake concentration on the validator/delegator side. The `TrackStakeChangesDecorator` ante handler rejects any transaction that would:

- move total bonded stake by more than 5% within a twelve-hour window, or
- give any single delegator more than 30% of total bonded stake (`ErrExceedsMaxStakeShare`).

Reporting power has no equivalent limit. A reporter's power is the sum of the bonded tokens of up to `max_selectors` (default 100) selectors, so a reporter can aggregate the stake of many delegators who are each individually under the 30% delegator cap. Nothing today stops a single reporter from accumulating 30%, 50%, or more of total reporting power. A reporter that large can dominate medians on low-participation queries, carries outsized weight in dispute voting (reporter group), and concentrates the impact of a single bad submission.

This ADR adds the same idea on the reporter side: **no single reporter may reach or exceed 30% of total reporting power on chain**, enforced by rejecting the transactions that would push a reporter over the line. The chain is assumed to be below the cap for every reporter at activation; the mechanism prevents crossings rather than remediating existing concentration.

### What counts as a reporter's power for the cap

For cap purposes we use a conservative "potential stake" for the reporter, not the exact stake a report would use at that instant:

- the bonded tokens of every selector currently selecting the reporter, **including** dispute-locked selectors (their stake returns when the lock expires), **excluding** selectors with a pending switch away (their stake already stopped counting and is committed elsewhere), plus
- the bonded tokens of selectors with a **pending switch into** the reporter (that stake lands at finalization, so it must be booked against the cap when the switch is scheduled, otherwise two concurrent inflows could each pass the check and overshoot together).

The denominator is total bonded tokens, the same quantity `TotalReporterPower()` already returns and the same denominator the delegator-side cap uses. Reporter stake is a subset of bonded tokens, so the ratio is well defined.

### Enforcement points

All enforcement lives in the existing `TrackStakeChangesDecorator` (x/reporter/ante), next to the delegator cap, so one decorator owns all stake-concentration limits and over-cap transactions are rejected at CheckTx before they enter the mempool. The decorator projects the final post-transaction state across all messages in the tx (consistent with how the 5% and delegator-30% checks already work), then runs the cap check once per affected reporter:

1. **MsgSelectReporter** — the selector's projected bonded stake joins the target reporter.
2. **MsgSwitchReporter** — same, against the destination reporter (checked at scheduling time; the stake actually lands at finalization). Re-sends of an already-pending switch to the same destination are not double-counted.
3. **MsgCreateReporter** — the creator's own projected bonded stake becomes the new reporter's power (both the fresh-create and selector-conversion paths).
4. **Staking messages** (`MsgDelegate`, `MsgBeginRedelegate`, `MsgCancelUnbondingDelegation`, `MsgCreateValidator`) — the decorator already computes per-delegator bonded deltas, including deltas caused by validators entering/leaving the active set within the tx. Each positive delta is attributed to the delegator's selected reporter (honoring selections made earlier in the same tx) and the affected reporter is re-checked.

A transaction is rejected with `ErrExceedsMaxReporterPower` if any affected reporter's projected potential stake would be **greater than or equal to** the cap fraction of projected total bonded stake. Note the boundary differs deliberately from the delegator cap (which rejects only strictly above 30%): the requirement here is that a reporter must never *reach* 30%.

Decreases are never blocked: a reporter already at/over the cap (possible only through passive drift, see below) can always shed stake, and its selectors can always undelegate or switch away.

### The cap is a module parameter

`max_reporter_power_share` (Dec) is added to x/reporter params, default `0.30`. A value of `1` or greater disables the check.

The delegator cap is hardcoded; this one cannot be, for a practical reason: the check fires on `CreateReporter`/`SelectReporter`, which every chain must execute during bootstrap. On a fresh chain with one validator, the validator holds ~100% of bonded stake and could never register a reporter — the oracle would be unusable on every devnet, local network, and young testnet. The same applies to the e2e suite, which routinely runs 2–3 validator chains where each validator holds 33–50% of bonded stake and registers itself as a reporter. Making the threshold a parameter keeps mainnet at a secure default while letting small networks raise or disable it explicitly in genesis (the standard e2e genesis disables it; the dedicated cap tests set it to 0.30 with a suitable validator distribution). Mainnet receives the default via the upgrade handler.

## Alternative Approaches

### Enforce in the x/reporter message handlers instead of ante

Handler checks see authoritative state and are simpler to write, but they split the concentration limits across two mechanisms (staking messages can only be intercepted in ante), they burn the user's fee on failure instead of rejecting at the mempool, and they cannot see the combined effect of multiple messages in one transaction the way the existing projection tracker does. Keeping everything in the one decorator that already owns stake-concentration policy was judged clearer.

Ante-only enforcement is sound only if no execution path runs messages without the ante chain. The one such path in the app was the ICA host, which executes ICA-relayed messages straight through the `MsgServiceRouter` — and mainnet's ICA host allowlist was found set to `["*"]` (verified against `mainnet.tellorlayer.com`), meaning ICA could bypass not just this cap but the pre-existing delegator cap and 5% tracker. Rather than duplicating every ante check into handlers, the v6.1.6 upgrade disables interchain accounts entirely (see Issues).

### Maintain a materialized per-reporter power total in state

A running tally updated by staking hooks would make the cap check O(1). It was rejected because reporter power is not an additive function of delegation events: it changes when validators enter or leave the bonded set, when selector locks expire, when switches finalize, and when reporters are jailed. The module already chose lazy recomputation with recalc flags (`ReporterStake`) instead of incremental maintenance for exactly this reason; a second, parallel incremental tally would be a standing source of consensus-risk bugs. The cap check instead recomputes the affected reporter's potential stake on demand, bounded by `max_selectors × max_num_of_delegations` (≤ ~1,000 store reads) and paid for by the transaction's gas.

### Cap effective power at report time instead of blocking acquisition

Clamping `ReporterStake` to 30% of total bonded at `SubmitValue` time would make the invariant unconditional (immune to all drift vectors below) without ever bricking a reporter. It was deferred, not rejected: it touches the oracle aggregation, reward-distribution period tracking, and dispute snapshot paths, and the power a report carries would diverge from the stake actually at risk behind it. It is the natural phase-2 defense-in-depth if drift past the cap is ever observed in practice. Rejecting reports outright from an over-cap reporter was rejected: a reporter can drift over the cap through no action of its own and must not lose the ability to operate.

### Hardcode 30% like the delegator cap

Breaks chain bootstrap and most of the existing test infrastructure, as described above.

## Issues / Notes on Implementation

- **The delegator cap stays hardcoded; the inconsistency is deliberate.** The reporter cap is a parameter and the delegator 30% cap remains hardcoded in the ante decorator. The two caps face different constraints: the delegator cap only blocks stake *increases*, so a chain whose genesis validators exceed 30% still bootstraps and operates; the reporter cap fires on `CreateReporter`/`SelectReporter`, which every chain must execute to have an oracle at all, so it cannot be hardcoded without killing small networks. Promoting the delegator cap to a matching `max_delegator_stake_share` param was considered (it would also let small devnets accept delegations to over-30% validators, which the hardcoded cap currently blocks) and deliberately deferred: it changes the mutability of an existing live limit from "chain upgrade required" to "governance vote", and that trade-off deserves its own decision rather than riding along here. Revisit if the devnet friction or the inconsistency becomes a problem.
- **Interchain accounts are disabled; interchain queries stay.** ICA-executed messages reach module handlers through the `MsgServiceRouter` without the ante chain, so an enabled ICA host with a permissive allowlist (mainnet had `allow_messages: ["*"]`) bypasses every ante-enforced limit: the 5% stake tracker, the 30% delegator cap, max delegations, and this reporter power cap. The v6.1.6 upgrade sets `host_enabled: false` with an empty allowlist and `controller_enabled: false`, and the app's default genesis ships both disabled, so new chains start safe. The async-ICQ module (used to serve oracle data to counterparty chains) does not execute messages and remains enabled. If ICA is ever wanted again, re-enabling via governance must come with a strict `allow_messages` list that excludes staking and reporter messages — or with these limits duplicated in message handlers.
- **Passive drift is not blocked.** A reporter's share can still reach 30% without any blockable transaction: total bonded stake shrinking (bounded by the existing 5%-per-12h tracker), validators entering/leaving the bonded set at end-block (jailing, slashing), dispute resolutions re-delegating returned stake, selectors' dispute locks expiring, and tip withdrawals delegating small amounts (`MsgWithdrawTip` performs a delegation outside the tracked staking messages — a pre-existing gap shared with the delegator cap, negligible in magnitude). Under the activation assumption (nobody at/over cap) plus the acquisition checks, drift past the cap requires the denominator to move against an already-near-cap reporter. If observed, phase 2 (report-time clamping) closes it.
- **Conservative overcounting is accepted.** Counting dispute-locked selectors and pending incoming switches means the check can reject a transaction even though the reporter's instantaneous reporting power is below the cap. This errs on the side of the invariant and avoids time-dependent loopholes (jail/lock windows as accumulation vehicles). Jailed reporters are checked the same as active ones for the same reason.
- **Gas cost.** Selecting to or delegating under a reporter with many selectors now performs a bounded selector scan in ante (comparable to what `SubmitValue` already does on every report). The scan consumes gas through normal store reads plus an explicit per-selector charge, mirroring the active-set scan precedent in the same decorator, so it cannot be used as a free-compute DoS vector.
- **Exact-boundary semantics.** `projected_reporter_stake * 1 >= max_reporter_power_share * projected_total_bonded` rejects. With the default, a reporter may hold at most one token-unit less than 30%.
- **Migration.** The new param deserializes as nil/zero for existing chains; the upgrade handler sets it to the 0.30 default. The ante check treats a nil/zero param (pre-upgrade state, or chains that never migrated) as disabled rather than as "cap everything at 0", which would halt all staking.
- **No retroactive remediation.** If a reporter is at/over the cap when the upgrade activates (contrary to the stated assumption), nothing forces divestment; the reporter simply cannot grow, and the existing paths (switch away, undelegate, RemoveSelector) remain available to shrink it.
