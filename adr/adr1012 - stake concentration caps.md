# ADR 1012: Stake concentration caps

## Authors

@danflo27

## Changelog

- 2026-06-12: initial version (reporter power cap)
- 2026-06-12: disable interchain accounts (host and controller) in the same upgrade after finding mainnet's ICA host allowlist set to `["*"]`; only interchain queries remain supported
- 2026-06-12: documented the decision to leave the delegator 30% cap hardcoded while the reporter cap is a param
- 2026-06-23: retitled to cover all stake-concentration limits in `TrackStakeChangesDecorator`; added validator power cap (`max_validator_power_share`)
- 2026-07-06: dispute execution now defers instead of failing BeginBlock when no viable validator can accept a slashed-token return; documented mixed-tx `MsgUnjail` projection behavior
- 2026-07-07: cap/dispute/validator-gap remediation: corrected escrow `Total` to coins actually collected, split `ReturnSlashedTokens` into collected principal plus explicit reporter-win `extraReturn`, replaced unbounded restorative bonding with headroom-bound returns plus unbonding overflow, made `CheckClosedDisputesForExecution` roll back and alert on any execution error instead of halting, replaced the 32-validator fallback scan with a full lowest-first scan via `GetBondedValidatorsByPower`, bounded execution to 1000 disputes per block, re-snapshotted vote denominators per dispute round and de-duplicated round ids, added bounded-window metrics, and documented residual risks.

## Context

Layer limits stake concentration through the `TrackStakeChangesDecorator` ante handler. It already rejects any transaction that would:

- move total bonded stake by more than 5% within a twelve-hour window, or
- give any single delegator more than 30% of total bonded stake (`ErrExceedsMaxStakeShare`).

Two gaps remain. **Reporting power** has no equivalent limit: a reporter's power is the sum of the bonded tokens of up to `max_selectors` (default 100) selectors, so a reporter can aggregate the stake of many delegators who are each individually under the 30% delegator cap. Nothing stops a single reporter from accumulating 30%, 50%, or more of total reporting power. A reporter that large can dominate medians on low-participation queries, carries outsized weight in dispute voting (reporter group), and concentrates the impact of a single bad submission.

**Validator bonded stake** is also uncapped. The controls above do not limit a single bonded validator's share of total bonded stake. A validator can acquire bonded stake up to 100% through paths that never trip the per-delegator cap or the 5% tracker:

- **Bonded-to-bonded redelegation**: moving stake from one bonded validator to another changes neither total bonded stake (5% tracker sees zero net delta) nor the redelegator's own bonded balance (per-delegator cap sees a zero delta). A destination validator can be pushed far past 30% this way.
- **Reporter keeper direct delegations**: `WithdrawTip`, `ReturnSlashedTokens`, `FeeRefund`, and `AddAmountToStake` call `stakingKeeper.Delegate` directly, with no staking tx message and therefore no ante stake-change accounting at all.

This ADR closes both gaps with governance-controlled reporter module params, each defaulting to `0.30`, enforced by rejecting transactions and bounding keeper return paths that would push a reporter or validator over the line. The chain is assumed to be below each cap at activation. The acquisition checks prevent crossings for tracked acquisition transactions, but they do not make crossing impossible: returned bonded value now bonds only up to active cap headroom and sends overflow through unbonding, while unbonded-stake preservation can later re-enter as bonded without immediate re-checking, governance proposals bypass ante by design, and passive denominator drift can push an existing position past the line. These residual vectors are accepted risks and are documented under **Issues / Notes on Implementation**.

## Reporter power cap

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

## Validator power cap

`max_validator_power_share` (Dec) is added to x/reporter params alongside `max_reporter_power_share`, not in `x/staking`, to avoid forking Cosmos SDK staking while exposing the value through existing reporter genesis, params query, and governance update paths.

- Default is `0.30`. Nil, zero, and values `>= 1` disable the check at enforcement sites. Proto uses `nullable=false`, so nil checks are defensive for pre-upgrade/malformed state.
- The boundary is strict `>`: a validator holding **exactly** the configured share is allowed. This deliberately differs from the reporter cap above, which rejects at `>=`.
- Enforcement is **acquisition-only**:
  - A validator that is already over cap may shrink or stay neutral.
  - A validator may become over cap passively because total bonded stake shrinks elsewhere (allowed denominator drift).
  - A validator may **not** gain active bonded stake while the resulting share is strictly above the cap.
- Entering the active set counts as acquisition: an unbonded/unbonding validator that transitions into the active bonded set is checked as `before = 0`, `after = postTokens`, including active-set replacement and `MsgUnjail` re-entry.

### Enforcement sites

- **Ante** (`TrackStakeChangesDecorator`): `MsgDelegate`, `MsgCreateValidator`, `MsgBeginRedelegate`, `MsgCancelUnbondingDelegation`, `MsgUndelegate` (via active-set replacement), authz-wrapped variants (respecting `MaxNestedMsgCount = 7`), and `MsgUnjail`.
- **Reporter keeper direct delegations**:
  - `WithdrawTip` hard-rejects the user-chosen over-cap destination; no scan fallback.
  - `ReturnSlashedTokens` preserves valid original delegation targets as the preferred restorative return destination, but only for the amount that fits under validator/delegator/reporter cap headroom. If the original validator is missing, invalid, or has no headroom, it falls back to a full scan of the active bonded set (lowest-power-first via `GetBondedValidatorsByPower`). Any amount that cannot safely bond returns through the staking unbonding queue rather than bonding over the cap or deferring execution. This BeginBlock path may append a UBD entry beyond the user-tx `MaxEntries` admission check as a narrow liveness exception.
  - `FeeRefund` and `AddAmountToStake` use the same headroom splitter and full lowest-first scan for bonded destinations. Cap overflow returns through unbonding, but these user-transaction paths still enforce the SDK `MaxEntries` admission check and may fail on that non-cap error.

### Resolved design points

- **Unjail isolation applies to pure-unjail transactions only**: a pure `MsgUnjail` does not set the active-set delta or touch `totalBondedDelta`, so it never routes through the 5% tracker, per-delegator cap, or reporter cap. The validator cap lazily projects the active-set re-entry for the unjailed validator only. If a tx **combines** `MsgUnjail` with any message that marks the active-set delta — including the common jail-recovery pairing `[MsgDelegate top-up + MsgUnjail]`, or any `MsgUndelegate`/`MsgBeginRedelegate` — the unjailed validator's projected re-entry is treated like any other active-set entry: its full stake folds into `totalBondedDelta` and the per-delegator deltas, so it **is** counted by the 5% tracker and the delegator/reporter caps. For a validator holding more than 5% of tracked total stake, the combined tx is rejected even though each message passes on its own. This is deliberate conservative accounting (an active-set entry is never under-counted once the projection machinery is engaged); the supported flow for large validators is to submit the top-up delegation and `MsgUnjail` as two separate transactions.
- **Headroom-bound returns replace the old restorative cap exemption**: restored slash principal, explicit reporter-win `extraReturn`, SUPPORT bond rewards, and from-bond fee refunds may bond only up to active validator/delegator/reporter cap headroom. Overflow returns through the staking unbonding queue, never liquid and never bonded over cap. Bonded originals remain the preferred destination for `ReturnSlashedTokens`; existing not-bonded originals still return with `tokenSrc = stakingtypes.Unbonded` because they do not immediately create active bonded stake.
- **Invalid exchange-rate destinations are never preserved**: validators with `InvalidExRate()` (`Tokens == 0` with positive `DelegatorShares`) cannot accept new delegation in the Cosmos SDK. `ReturnSlashedTokens`, `FeeRefund`, and `AddAmountToStake` skip these validators before calling `Delegate`; if an original dispute-return destination is invalid, the keeper falls back to the bonded scan.
- **Full active-set scan for fallback destinations**: automatic fallback destinations query the entire active bonded set via `GetBondedValidatorsByPower` (bounded by `MaxValidators`) and iterate it lowest-power-first to find the first viable validator with positive cap headroom and valid SDK exchange-rate state. The local `GetBondedValidators` helper and the `ValidatorPowerFallbackScanLimit = 32` constant were removed. If only partial headroom exists, only that partial amount bonds and the overflow unbonds; an empty bonded set returns `ErrExceedsMaxValidatorPowerShare` rather than panicking.
- **Fallback exhaustion must not halt the chain**:
  - `WithdrawTip` remains hard-rejected for a user-chosen over-cap destination. `FeeRefund` and `AddAmountToStake` no longer fail solely because cap headroom is unavailable: they bond available headroom and return cap overflow through unbonding. They still fail on non-cap errors, including `ErrDelegatorShareExRateInvalid` and `ErrMaxUnbondingDelegationEntries`.
  - `ReturnSlashedTokens` runs inside the dispute module's BeginBlocker (`ExecuteVote`). `CheckClosedDisputesForExecution` now runs each `ExecuteVote` against a cache context and rolls back **all** errors, not only cap/exrate errors: the dispute stays `PendingExecution`, no partial state commits, and BeginBlock returns `nil`. Retryable stake-distribution errors emit a `dispute_execution_deferred` event (Info log); any other error emits a `dispute_execution_failed` event (Error log) and a telemetry counter so operators are paged.
  - For cap/removed-validator restoration, `ReturnSlashedTokens` places overflow in the staking unbonding queue against the original validator address rather than returning a cap error. This is safe even if the validator has been removed (`SetUnbondingDelegationEntry` and `CompleteUnbonding` do not require the validator to exist), and the slashing `AfterUnbondingInitiated` hook is a no-op in the SDK version used.
  - If no slash principal was collected (`snapshot.Total == 0`) but a reporter-win `extraReturn` is positive, `ReturnSlashedTokens` refuses to invent a recipient or spend fee funds without an origin basis and returns a distinct `ErrReturnExtraWithoutPrincipal`; the P1-2 rollback/alert path pages operators instead of silently deferring.
  - A dispute that fails identically every block with a non-cap error will retry-and-alert indefinitely. A future `MaxDeferralBlocks` auto-resolve circuit breaker for the non-cap path is the documented future direction; it is not implemented in this remediation.

### Non-acquisition paths not rejected by this cap

- `MsgEditValidator` commission/metadata changes (no token or bonded-share movement).
- `ReturnSlashedTokens` preserving an existing not-bonded original validator with `tokenSrc = stakingtypes.Unbonded`.
- Governance changes to `MaxValidators` that shrink the active set (passive exits / denominator drift).
- Validators skipped by `TokensToConsensusPower(postTokens, powerReduction) == 0` in active-set projection.
- `MsgUnjail` for a validator that does not make the top set.
- Already-over-cap validators undelegating or redelegating away.

This cap is separate from the reporter power cap (which limits reporter *potential* stake and rejects at `>=`) and from the 12-hour 5% validator-set movement throttle. It specifically closes bonded-to-bonded validator concentration acquisition and the direct-keeper delegation bypasses.

## Alternative Approaches

### Enforce in the x/reporter message handlers instead of ante

Handler checks see authoritative state and are simpler to write, but they split the concentration limits across two mechanisms (staking messages can only be intercepted in ante), they burn the user's fee on failure instead of rejecting at the mempool, and they cannot see the combined effect of multiple messages in one transaction the way the existing projection tracker does. Keeping everything in the one decorator that already owns stake-concentration policy was judged clearer.

Ante-only enforcement is sound only if no execution path runs messages without the ante chain. The one such path in the app was the ICA host, which executes ICA-relayed messages straight through the `MsgServiceRouter` — and mainnet's ICA host allowlist was found set to `["*"]` (verified against `mainnet.tellorlayer.com`), meaning ICA could bypass not just these caps but the pre-existing delegator cap and 5% tracker. Rather than duplicating every ante check into handlers, the v6.1.6 upgrade disables interchain accounts entirely (see Issues).

### Maintain a materialized per-reporter power total in state

A running tally updated by staking hooks would make the cap check O(1). It was rejected because reporter power is not an additive function of delegation events: it changes when validators enter or leave the bonded set, when selector locks expire, when switches finalize, and when reporters are jailed. The module already chose lazy recomputation with recalc flags (`ReporterStake`) instead of incremental maintenance for exactly this reason; a second, parallel incremental tally would be a standing source of consensus-risk bugs. The cap check instead recomputes the affected reporter's potential stake on demand, bounded by `max_selectors × max_num_of_delegations` (≤ ~1,000 store reads) and paid for by the transaction's gas.

### Cap effective power at report time instead of blocking acquisition

Clamping `ReporterStake` to 30% of total bonded at `SubmitValue` time would make the invariant unconditional (immune to all drift vectors below) without ever bricking a reporter. It was deferred, not rejected: it touches the oracle aggregation, reward-distribution period tracking, and dispute snapshot paths, and the power a report carries would diverge from the stake actually at risk behind it. It is the natural phase-2 defense-in-depth if drift past the cap is ever observed in practice. Rejecting reports outright from an over-cap reporter was rejected: a reporter can drift over the cap through no action of its own and must not lose the ability to operate.

### Hardcode 30% like the delegator cap

Breaks chain bootstrap and most of the existing test infrastructure, as described above.

## Issues / Notes on Implementation

- **The delegator cap stays hardcoded; the inconsistency is deliberate.** The reporter and validator caps are parameters and the delegator 30% cap remains hardcoded in the ante decorator. The reporter cap faces a bootstrap constraint (`CreateReporter`/`SelectReporter` must succeed on small networks); the validator cap shares the same param pattern for governance/genesis flexibility. Promoting the delegator cap to a matching `max_delegator_stake_share` param was considered (it would also let small devnets accept delegations to over-30% validators, which the hardcoded cap currently blocks) and deliberately deferred: it changes the mutability of an existing live limit from "chain upgrade required" to "governance vote", and that trade-off deserves its own decision rather than riding along here. Revisit if the devnet friction or the inconsistency becomes a problem.
- **Interchain accounts are disabled; interchain queries stay.** ICA-executed messages reach module handlers through the `MsgServiceRouter` without the ante chain, so an enabled ICA host with a permissive allowlist (mainnet had `allow_messages: ["*"]`) bypasses every ante-enforced limit: the 5% stake tracker, the 30% delegator cap, max delegations, the reporter power cap, and the validator power cap. The v6.1.6 upgrade sets `host_enabled: false` with an empty allowlist and `controller_enabled: false`, and the app's default genesis ships both disabled, so new chains start safe. The async-ICQ module (used to serve oracle data to counterparty chains) does not execute messages and remains enabled. If ICA is ever wanted again, re-enabling via governance must come with a strict `allow_messages` list that excludes staking and reporter messages — or with these limits duplicated in message handlers.
- **Passive drift is not blocked (reporter cap).** A reporter's share can still reach 30% without any blockable transaction: total bonded stake shrinking (bounded by the existing 5%-per-12h tracker), validators entering/leaving the bonded set at end-block (jailing, slashing), dispute resolutions re-delegating returned stake, selectors' dispute locks expiring, and tip withdrawals delegating small amounts (`MsgWithdrawTip` performs a delegation outside the tracked staking messages — a pre-existing gap shared with the delegator cap, negligible in magnitude). Under the activation assumption (nobody at/over cap) plus the acquisition checks, drift past the cap requires the denominator to move against an already-near-cap reporter. If observed, phase 2 (report-time clamping) closes it.
- **Conservative overcounting is accepted (reporter cap).** Counting dispute-locked selectors and pending incoming switches means the check can reject a transaction even though the reporter's instantaneous reporting power is below the cap. This errs on the side of the invariant and avoids time-dependent loopholes (jail/lock windows as accumulation vehicles). Jailed reporters are checked the same as active ones for the same reason.
- **Gas cost (reporter cap).** Selecting to or delegating under a reporter with many selectors now performs a bounded selector scan in ante (comparable to what `SubmitValue` already does on every report). The scan consumes gas through normal store reads plus an explicit per-selector charge, mirroring the active-set scan precedent in the same decorator, so it cannot be used as a free-compute DoS vector.
- **Exact-boundary semantics.** Reporter: `projected_reporter_stake * 1 >= max_reporter_power_share * projected_total_bonded` rejects; with the default, a reporter may hold at most one token-unit less than 30%. Validator and delegator: strict `>` - exactly 30% is allowed. This asymmetry is intentional.
- **Migration.** Both params deserialize as nil/zero for existing chains; the upgrade handler sets each to the 0.30 default. The ante check treats a nil/zero param (pre-upgrade state, or chains that never migrated) as disabled rather than as "cap everything at 0", which would halt all staking.
- **No retroactive remediation (reporter cap).** If a reporter is at/over the cap when the upgrade activates (contrary to the stated assumption), nothing forces divestment; the reporter simply cannot grow, and the existing paths (switch away, undelegate, RemoveSelector) remain available to shrink it.
- **Governance / genesis / upgrade (validator cap).** Governance, genesis, and upgrade parameter changes are trusted configuration paths. They may disable or tighten the validator cap but do not themselves execute active bonded stake acquisition.
- **Residual risks accepted by this remediation.**
  - **Unbonded preserve → bulk bond on re-entry.** A not-bonded original validator is preserved as `tokenSrc = Unbonded` and is not cap-checked. If that validator later bonds, the restored stake enters the active set en masse without an acquisition-time cap check. This is the ADR's already-accepted residual risk; the phase-2 report-time clamp is the long-term bound.
  - **Governance-proposal ante bypass.** Staking/reporter messages executed by a passed governance proposal bypass the ante decorator. A passed proposal is chain-level consent and can already change cap params directly, so this is accepted; the reviewer-checklist lint helps prevent new un-cap-checked keeper `Delegate` sites.
  - **Passive drift bounded only by the phase-2 clamp.** Denominator shrink, validator set churn, jail/unjail, dispute restoration, and small tip delegations can still move a party past the cap. The phase-2 report-time clamp is the intended long-term bound.
  - **UBD `MaxEntries` liveness exception.** The BeginBlock-only restoration overflow path calls `SetUnbondingDelegationEntry` directly and therefore does not enforce the user-tx `MaxEntries` admission check. This avoids keeping INVALID/AGAINST dispute execution pending forever, never creates bonded concentration, and the extra entry still matures through the normal UBD queue. User-transaction overflow paths (`FeeRefund`, `AddAmountToStake`) enforce `HasMaxUnbondingDelegationEntries` and may fail with `ErrMaxUnbondingDelegationEntries`.
  - **Zero-collected principal + positive reporter-win extra return.** If no slash principal was escrowed but an AGAINST dispute would return fee upside, `ReturnSlashedTokens` returns `ErrReturnExtraWithoutPrincipal` and the P1-2 alert path fires. There is no auto-resolve; operators must investigate.
  - **Non-cap execution errors retry forever.** The P1-2 halt guard rolls back and alerts on any unexpected `ExecuteVote` error, but there is no circuit breaker. A permanently-failing dispute emits `dispute_execution_failed` every block until fixed or until a future `MaxDeferralBlocks` policy is adopted.
- **`ReturnSlashedTokens` extra-return semantics.** INVALID / NO_QUORUM_MAJORITY_INVALID returns only the actually escrowed slash principal (`extraReturn = 0`). AGAINST / NO_QUORUM_MAJORITY_AGAINST returns the escrowed principal plus the dispute fee minus its 5% burn, passed explicitly as `extraReturn`; `SlashAmount` is never mutated. Fee funds are therefore never used to cover phantom slash principal.
- **Round-2+ dispute fees are non-refundable.** `AddDisputeRound` consumes later-round fees into `BurnAmount` (half burned, half voter reward). Only the round-1 payer has a refund path via `WithdrawFeeRefund`; escalating parties knowingly commit those fees.
- **Round-2+ disputes do not re-slash.** `AddDisputeRound` intentionally does not call `SlashAndJailReporter`/`EscrowReporterStake` again. Escalation fees are the round-N economic commitment; re-slashing per round would double-punish before a verdict.
- **SUPPORT bond rewards and from-bond fee refunds use bond-to-headroom plus unbond-overflow.** `WithdrawFeeRefund` (SUPPORT bond reward via `AddAmountToStake`) and from-bond fee refund via `FeeRefund` are user transactions. They no longer fail solely because cap headroom is unavailable, but they still fail on non-cap staking admission errors such as `ErrMaxUnbondingDelegationEntries`.
- **FeeRefund vs ReturnSlashedTokens pool divergence.** `FeeRefund` forces a bonded destination preference for fee refunds, while `ReturnSlashedTokens` preserves the original pool preference (bonded or unbonded). This is intentional: restored slash principal reverses the original delegation; fee refunds are a new bonded acquisition. Both paths share the same cap-overflow rule: bond only cap-safe headroom and send overflow through unbonding.
- **Per-round vote denominator re-snapshot and round-id de-duplication.** `AddDisputeRound` calls `SetBlockInfo(ctx, dispute.HashId)` so the new round tallies against fresh `TotalReporterPower`/`TotalUserTips`. `GetSumOfUserAndReporterVotesAllRounds` de-duplicates round ids so `PrevDisputeIds` (which includes the current id) is not double-counted.
- **Bounded-window BeginBlock metrics.** `CheckClosedDisputesForExecution` processes at most 1000 pending disputes per block. Telemetry gauges/counters report counts and ages over the inspected window only, not a global unbounded scan.

## Reviewer checklist

Any future direct reporter/dispute staking delegation must use `CheckValidatorPowerShareDelegation` (hard-reject user paths) or `returnStakeAmount` / `delegateBondedWithOverflow` (cap-aware bonded restoration with unbonding overflow), unless explicitly documented as non-bonded/non-acquisition:

```bash
rg -n 'stakingKeeper\.Delegate|\.Delegate\(ctx' x/reporter x/dispute
```
