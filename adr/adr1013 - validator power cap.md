# ADR 1013: Validator power cap

## Authors

@danflo27

## Changelog

- 2026-06-23: initial version

## Context

Layer's existing stake-concentration controls (ADR 1012 and the `TrackStakeChangesDecorator` ante handler) limit total bonded-stake movement (5% / 12h) and per-delegator bonded stake (30%). They do **not** cap a single bonded validator's share of total bonded stake. A validator can acquire bonded stake up to 100% through paths that never trip the per-delegator cap or the 5% tracker:

- **Bonded-to-bonded redelegation**: moving stake from one bonded validator to another changes neither total bonded stake (5% tracker sees zero net delta) nor the redelegator's own bonded balance (per-delegator cap sees a zero delta). A destination validator can be pushed far past 30% this way.
- **Reporter keeper direct delegations**: `WithdrawTip`, `ReturnSlashedTokens`, `FeeRefund`, and `AddAmountToStake` call `stakingKeeper.Delegate` directly, with no staking tx message and therefore no ante stake-change accounting at all.

This ADR closes that acquisition gap with a governance-controlled reporter module param, `max_validator_power_share` (default `0.30`), that rejects execution paths increasing a validator's **active bonded stake** above the configured share of total bonded stake.

## Decision

- `max_validator_power_share` lives in `x/reporter` params (alongside `max_reporter_power_share`), not in `x/staking`, to avoid forking Cosmos SDK staking while exposing the value through existing reporter genesis, params query, and governance update paths.
- Default is `0.30`. Nil, zero, and values `>= 1` disable the check at enforcement sites. Proto uses `nullable=false`, so nil checks are defensive for pre-upgrade/malformed state.
- The boundary is strict `>`: a validator holding **exactly** the configured share is allowed. This deliberately differs from the reporter cap (ADR 1012), which rejects at `>=`.
- Enforcement is **acquisition-only**:
  - A validator that is already over cap may shrink or stay neutral.
  - A validator may become over cap passively because total bonded stake shrinks elsewhere (allowed denominator drift).
  - A validator may **not** gain active bonded stake while the resulting share is strictly above the cap.
- Entering the active set counts as acquisition: an unbonded/unbonding validator that transitions into the active bonded set is checked as `before = 0`, `after = postTokens`, including active-set replacement and `MsgUnjail` re-entry.

### Enforcement sites

- **Ante** (`TrackStakeChangesDecorator`): `MsgDelegate`, `MsgCreateValidator`, `MsgBeginRedelegate`, `MsgCancelUnbondingDelegation`, `MsgUndelegate` (via active-set replacement), authz-wrapped variants (respecting `MaxNestedMsgCount = 7`), and `MsgUnjail`.
- **Reporter keeper direct delegations**: `WithdrawTip` (hard-rejects the user-chosen over-cap destination, no scan fallback), and `ReturnSlashedTokens` / `FeeRefund` / `AddAmountToStake` (use a bounded, deterministic scan of bonded validators in power-store order to pick the first under-cap destination; never split amounts).

### Resolved design points

- **M2 — Unjail isolation**: a pure `MsgUnjail` does not set the active-set delta or touch `totalBondedDelta`, so it never routes through the 5% tracker, per-delegator cap, or reporter cap. The validator cap lazily projects the active-set re-entry for the unjailed validator only.
- **M4 — `ReturnSlashedTokens` unbonded destination gap**: if the original validator exists but is not bonded, restored stake is delegated with `tokenSrc = stakingtypes.Unbonded` and is intentionally **not** cap-checked, because it does not immediately create active bonded stake. This is an accepted residual risk documented in the implementation plan.
- **M5 — Bounded deterministic scan, no exemption**: automatic fallback destinations scan a bounded list (`ValidatorPowerFallbackScanLimit = 32`) of bonded validators in power-store order and pick the first that passes the cap. No amount splitting, no unbounded scan, no dispute-only exemption. An empty bonded set or scan exhaustion returns `ErrExceedsMaxValidatorPowerShare` rather than panicking.

## Non-acquisition paths not rejected by this cap

- `MsgEditValidator` commission/metadata changes (no token or bonded-share movement).
- Governance changes to `MaxValidators` that shrink the active set (passive exits / denominator drift).
- Validators skipped by `TokensToConsensusPower(postTokens, powerReduction) == 0` in active-set projection.
- `MsgUnjail` for a validator that does not make the top set.
- Already-over-cap validators undelegating or redelegating away.

## Distinction from ADR 1012 / the 5% tracker

This cap is separate from ADR 1012's reporter power cap (which limits reporter *potential* stake and rejects at `>=`) and from the 12-hour 5% validator-set movement throttle. It specifically closes bonded-to-bonded validator concentration acquisition and the direct-keeper delegation bypasses.

## Governance / genesis / upgrade

Governance, genesis, and upgrade parameter changes are trusted configuration paths. They may disable or tighten the cap but do not themselves execute active bonded stake acquisition. The v6.1.6 upgrade handler sets the `0.30` default for chains whose pre-upgrade state deserializes the field as nil/zero.

## Reviewer checklist

Any future direct reporter/dispute staking delegation must use `CheckValidatorPowerShareDelegation`, `bondedValidatorForDelegation`, or `pickUnderCapBondedValidator`, unless explicitly documented as non-bonded/non-acquisition:

```bash
rg -n 'stakingKeeper\.Delegate|\.Delegate\(ctx' x/reporter x/dispute
```
