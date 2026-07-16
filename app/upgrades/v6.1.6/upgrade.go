package v6_1_6

import (
	"context"
	"fmt"

	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

/*
Upgrade to v6.1.6:
- Deferred reporter switches stored in OutgoingPendingSwitches / IncomingPendingSwitchIdx
  with ReporterPendingSwitchHeads for O(1) checks.
- Finalization runs at the start of ReporterStake (e.g. when a reporter submits),
  not in BeginBlock.
- Pending switch targets live only in keeper collections (not on Selection).
  Max pending switches per reporter is a module param (default 10).
- Reporter power cap (ADR 1012): new reporter module param max_reporter_power_share
  caps a single reporter's potential stake below a share of total bonded tokens
  (default 30%). Enforcement happens in the TrackStakeChangesDecorator ante handler
  on CreateReporter/SelectReporter/SwitchReporter and on staking messages that
  increase a selector's bonded stake. The param deserializes as nil for existing
  chains, which the ante treats as disabled; this handler sets the 0.30 default so
  the cap activates at upgrade.
- Validator power cap (ADR 1012): new reporter module param
  max_validator_power_share caps a single bonded validator's active bonded stake
  below a share of total bonded tokens (default 30%). Enforcement is
  acquisition-only and covers staking/authz messages in the
  TrackStakeChangesDecorator ante handler (including MsgUnjail active-set
  re-entry) and reporter keeper direct delegations (WithdrawTip,
  ReturnSlashedTokens, FeeRefund, AddAmountToStake). This handler sets the 0.30
  default so the cap activates at upgrade.
- Interchain accounts are disabled entirely (host and controller). Mainnet's
  ICA host allowed all messages, and ICA-executed messages go through the
  MsgServiceRouter without the ante chain, bypassing the stake and reporter
  power limits. Only interchain queries remain supported.
- Tip unlock-to-balance: ensure tips_unlock_pool module account for timed withdraw
  of tip rewards to free-floating balance (existing tip collections stay empty).
  Uses GetAccount + convert-or-create so a pre-upgrade BaseAccount at the
  deterministic module address cannot panic GetModuleAccount and halt consensus.
- x/group is removed (module and store key). Group proposal execution bypasses
  ante the same way ICA did. The old store is omitted from committed state; no
  deletion migration is configured, and the upgraded app does not mount its key.

No custom state migration is required beyond RunMigrations and module-account
creation: new collections and proto fields deserialize to empty / zero for
existing chains.
*/

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ak authkeeper.AccountKeeper,
	rk reporterkeeper.Keeper,
	ick icacontrollerkeeper.Keeper,
	ihk icahostkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))

		EnsureTipsUnlockPoolModuleAccount(sdkCtx, ak)

		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		params, err := rk.Params.Get(ctx)
		if err != nil {
			return vm, fmt.Errorf("reporter params: %w", err)
		}
		changed := false
		if params.MaxPendingSwitchesPerReporter == 0 {
			params.MaxPendingSwitchesPerReporter = reportertypes.DefaultMaxPendingSwitchesPerReporter
			changed = true
			sdkCtx.Logger().Info(
				"set reporter max_pending_switches_per_reporter",
				"value", params.MaxPendingSwitchesPerReporter,
			)
		}
		if params.MaxReporterPowerShare.IsNil() || params.MaxReporterPowerShare.IsZero() {
			params.MaxReporterPowerShare = reportertypes.DefaultMaxReporterPowerShare
			changed = true
			sdkCtx.Logger().Info(
				"set reporter max_reporter_power_share",
				"value", params.MaxReporterPowerShare.String(),
			)
		}
		if params.MaxValidatorPowerShare.IsNil() || params.MaxValidatorPowerShare.IsZero() {
			params.MaxValidatorPowerShare = reportertypes.DefaultMaxValidatorPowerShare
			changed = true
			sdkCtx.Logger().Info(
				"set reporter max_validator_power_share",
				"value", params.MaxValidatorPowerShare.String(),
			)
		}
		if changed {
			if err := rk.Params.Set(ctx, params); err != nil {
				return vm, fmt.Errorf("set reporter params: %w", err)
			}
		}

		ihk.SetParams(sdkCtx, icahosttypes.Params{HostEnabled: false, AllowMessages: []string{}})
		sdkCtx.Logger().Info("disabled interchain accounts host")
		ick.SetParams(sdkCtx, icacontrollertypes.Params{ControllerEnabled: false})
		sdkCtx.Logger().Info("disabled interchain accounts controller")
		sdkCtx.Logger().Info("removed x/group module")

		return vm, nil
	}
}

// EnsureTipsUnlockPoolModuleAccount creates tips_unlock_pool as a module account,
// or converts a non-module account already occupying that address (e.g. from a
// dust send on the pre-upgrade binary, when the address was unblocked).
//
// Do not call GetModuleAccount here: if a BaseAccount exists at the address it
// panics with "account is not a module account", which is unrecovered in the
// upgrade BeginBlocker.
func EnsureTipsUnlockPoolModuleAccount(sdkCtx sdk.Context, ak authkeeper.AccountKeeper) {
	name := reportertypes.TipsUnlockPool
	addr := authtypes.NewModuleAddress(name)
	acc := ak.GetAccount(sdkCtx, addr)
	if acc == nil {
		macc := authtypes.NewEmptyModuleAccount(name)
		maccI := ak.NewAccount(sdkCtx, macc).(sdk.ModuleAccountI)
		ak.SetModuleAccount(sdkCtx, maccI)
		sdkCtx.Logger().Info(
			"Created tips_unlock_pool module account",
			"address", addr.String(),
		)
		return
	}
	if _, ok := acc.(sdk.ModuleAccountI); ok {
		sdkCtx.Logger().Info("tips_unlock_pool module account already exists")
		return
	}

	// Preserve account number (and sequence). Clear pubkey — module accounts
	// do not support signing keys.
	baseAcc := authtypes.NewBaseAccount(acc.GetAddress(), nil, acc.GetAccountNumber(), acc.GetSequence())
	macc := authtypes.NewModuleAccount(baseAcc, name)
	ak.SetModuleAccount(sdkCtx, macc)
	sdkCtx.Logger().Info(
		"Converted existing account to tips_unlock_pool module account",
		"address", addr.String(),
		"account_number", acc.GetAccountNumber(),
	)
}
