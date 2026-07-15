# Implementation prompt: Withdraw tips to free-floating balance (timed unlock)

Give this document to an implementing agent. Follow these product decisions as written; do not invent partial-withdraw or amount-parameter UX.

---

## Background / why

Tips are paid from liquid TRB. After a report, tip rewards sit in `tips_escrow_pool` and accrue in `SelectorTips`. Today `MsgWithdrawTip` moves **all** claimable tips into stake (delegate to a bonded validator). Liquidity only returns after undelegate + staking unbonding (~21 days).

That stake hop exists so tip tokens cannot be immediately recycled into new tips (vote-farming / tipper-power farming), together with the 2% tip burn (see `adr/adr1005 - handling of tips after report.md`).

We want a second destination: withdraw tips toward free-floating balance, still delayed by the staking **unbonding period**, without forcing a temporary bond.

---

## Goals

1. Allow selectors to start a withdrawal of their **entire** available tip balance to liquid TRB after a timed escrow.
2. Escrow duration = `stakingKeeper.UnbondingTime(ctx)` (same length as stake unbonding; do not hardcode 21 days).
3. Process maturities in the **reporter EndBlocker** the same way staking processes mature unbondings: time-keyed queue, dequeue all mature entries and pay out.
4. Keep existing `WithdrawTip` → stake path unchanged.
5. Allow cancel of a **specific** in-flight unlock (by `unlock_id`): tokens return to `tips_escrow_pool` and become withdrawable again (to stake or to balance).
6. **No partial withdrawals** and **no partial cancels** (cancel returns that entry’s full amount only).

---

## Non-goals

- Partial amount args on withdraw or cancel.
- Changing the 2% tip burn or tipper voting-power accounting.
- Auto-delegate / auto-undelegate shim (do not route through bonded/not-bonded pools for this path).
- Vesting accounts.

---

## Module accounts

| Account | Role |
|---------|------|
| `tips_escrow_pool` (existing) | Claimable tip rewards (`SelectorTips`). Stake withdraw and “start unlock” both draw from here. |
| `tips_unlock_pool` (**new**) | Tokens mid-unlock after withdraw-to-balance. Only maturity payout or cancel may leave this pool. |

Register `tips_unlock_pool` next to `tips_escrow_pool` in app module accounts (`app/app.go` / genesis module account wiring).

### Balance invariants (should hold after every successful tx / end-block payout)

- `bank(tips_escrow_pool)` ≈ sum of all `SelectorTips` (truncated to int, same as today’s withdraw accounting).
- `bank(tips_unlock_pool)` ≈ sum of all in-flight `TipUnlock` entry amounts.

---

## User flows

### A. Withdraw to stake (existing — unchanged)

`MsgWithdrawTip(selector, validator)`

1. Settle pending distribution if needed (`SettleReporter`).
2. Take **all** `SelectorTips` (truncate int; remainder dust stays or is cleared per existing logic).
3. Delegate to bonded validator; move coins `tips_escrow_pool` → bonded pool (current behavior).

### B. Withdraw to balance (new)

Suggested: `MsgWithdrawTipToBalance(selector)` (name flexible; keep reporter module).

1. Settle pending distribution if needed (same as A).
2. Read **entire** available `SelectorTips`. If truncated amount is zero → error (`no tips to withdraw`).
3. Clear / reduce `SelectorTips` exactly as stake withdraw does today (full withdraw of available tips; preserve dust handling consistently with A).
4. `completion_time = ctx.BlockTime().Add(UnbondingTime)`.
5. Write unlock state + maturity queue entry (see Collections).
6. Bank: `SendCoinsFromModuleToModule(tips_escrow_pool → tips_unlock_pool, amount)`.
7. Emit event e.g. `tip_unlock_started` with selector, amount, completion_time, unlock_id.

**No amount field. No validator field.**

### C. Cancel unlock-to-balance (new)

`MsgCancelTipUnlock(selector, unlock_id)`

Cancel targets one in-flight entry by `unlock_id` (selectors may have multiple pending unlocks). Cancel is all-or-nothing for that entry’s full amount; send those tokens back to escrow.

1. Load `TipUnlocks` at `(selector, unlock_id)`.
2. If missing → error. (If EndBlocker already paid it out, the entry is gone → same error.)
3. Delete `TipUnlocks` row + corresponding `TipUnlockQueue` entry.
4. Bank: `SendCoinsFromModuleToModule(tips_unlock_pool → tips_escrow_pool, amount)`.
5. Credit `SelectorTips` by that amount (so they can call A or B again).
6. Emit e.g. `tip_unlock_cancelled` with selector, unlock_id, amount.

Cancel must **not** send coins to the user account. Timer is not “kept”; a later B starts a fresh unbonding-length clock.

### D. Maturity processing (reporter EndBlocker)

Handle maturities in the **reporter EndBlocker**, mirroring how the staking module dequeues mature unbondings (UBD queue):

1. Walk time-keyed `TipUnlockQueue` for all entries with `completion_time <= ctx.BlockTime()`.
2. For each: `SendCoinsFromModuleToAccount(tips_unlock_pool → selector, amount)`.
3. Delete unlock entry + queue key.
4. Emit e.g. `tip_unlock_completed`.

Process in deterministic key order. Drain all mature entries each block unless gas risk is proven; only then add a batch bound.

---

## Collections to add

Add prefixes after the current last reporter prefix (`SelectorDisputeLockPrefix = 35`).

| Collection | Type | Key | Value | Purpose |
|------------|------|-----|-------|---------|
| `TipUnlocks` | `Map` | `Pair(selector_addr, unlock_id)` | `TipUnlockEntry { amount, completion_time }` | Source of truth for in-flight unlocks; prefix-walk by selector for efficient per-address queries |
| `TipUnlockQueue` | `Map` | `Pair(completion_unix, unlock_id)` | `selector_addr` (bytes) or empty + rely on id join | EndBlocker maturity scan like UBD queue |
| `TipUnlockID` | `Sequence` or `Item[uint64]` | — | next id | Unique ids for concurrent unlocks |

On start (B): allocate id, set both maps.  
On cancel (C) / mature (D): remove both maps for that id.

### Proto sketch

```protobuf
message TipUnlockEntry {
  string amount = 1;                      // math.Int
  google.protobuf.Timestamp completion_time = 2;
}

// MsgWithdrawTipToBalance { string selector_address = 1; }
// MsgCancelTipUnlock {
//   string selector_address = 1;
//   uint64 unlock_id = 2;  // required: cancel one specific pending unlock
// }
```

Include genesis import/export for the new collections if the module exports other maps today.

---

## Full withdraw / cancel semantics (explicit)

- **Available tips** means current `SelectorTips` after settle — same pile A uses today.
- Multiple concurrent unlocks per selector are **allowed** (each B creates a new `unlock_id`). Starting B only pulls from claimable `SelectorTips` / `tips_escrow_pool`, never from `tips_unlock_pool`.
- No partials: at any moment claimable tips go 100% to stake **or** 100% into one new unlock entry. Cancel returns that entry’s full amount to escrow; user then chooses A or B again on the combined claimable balance.
- Dust / `LegacyDec` remainder in `SelectorTips`: match existing `WithdrawTip` truncate behavior exactly.

---

## Queries / CLI

- Query pending unlocks for a selector (list `TipUnlockEntry`s via prefix iterate on `TipUnlocks`).
- Optional: query unlock pool balance (bank) for ops.
- Autocli:
  - `withdraw-tip-to-balance [selector-address]`
  - `cancel-tip-unlock [selector-address] [unlock-id]`
- Keep existing `withdraw-tip [selector] [validator]`.

Update `x/reporter/readme.md` briefly.

---

## Security properties to preserve

1. Tokens from B are not tippable until maturity payout (they sit in `tips_unlock_pool`, not user balance).
2. Cancel restores escrow only — still not tippable until a future tip msg from liquid funds after a later maturity, or until user gets liquid another way.
3. Do not credit oracle `TipperTotal` on withdraw/cancel/mature.
4. Unlock duration always read from `stakingKeeper.UnbondingTime` at **start** time (completion frozen on the entry); later param changes do not rewrite existing completion times.

---

## Implementation checklist

1. Proto + codec + Msg server handlers.
2. Register `tips_unlock_pool` module account.
3. Collections + keeper wiring + genesis.
4. `WithdrawTipToBalance` + `CancelTipUnlock`.
5. EndBlocker mature processing (staking UBD-style queue walk).
6. Events + telemetry analogous to `tip_withdrawn`.
7. Unit tests: start unlock (bank moves + state), cancel (pool round-trip + SelectorTips restore), maturity (payout + deletes), zero tips error, double-cancel error.
8. Integration test: tip → report → withdraw-to-balance → advance time by UnbondingTime → balance credited; cancel path returns to escrow then stake withdraw still works.
9. Mock `StakingKeeper.UnbondingTime` in unit tests (already on reporter expected keepers).

---

## Locked decisions

| Topic | Decision |
|-------|----------|
| Multiple concurrent unlocks per selector | **Allow** (unique `unlock_id`). |
| Cancel targeting | **`MsgCancelTipUnlock` requires `unlock_id`** so a user can cancel one specific pending unlock when several are open. |
| Partial withdraw / partial cancel | **Never**. |
| Dust / LegacyDec remainder in `SelectorTips` | **Match existing `WithdrawTip` truncate behavior exactly.** |
| Where to process maturities | **Reporter EndBlocker**, time-keyed queue handled like staking unbonding maturity (UBD dequeue). |
| Pool name | **`tips_unlock_pool`**. |

---

## Reference code

- Existing stake withdraw: `x/reporter/keeper/msg_server.go` → `WithdrawTip`
- Tip escrow funding: `x/oracle/keeper/aggregate.go` (oracle → `tips_escrow_pool`)
- ADR intent: `adr/adr1005 - handling of tips after report.md`
- Unbonding duration dependency already used elsewhere: `stakingKeeper.UnbondingTime` (`x/reporter/types/expected_keepers.go`)
- Similar timed queues in reporter: `DistributionQueue`, pending-switch collections in `x/reporter/keeper/keeper.go`
