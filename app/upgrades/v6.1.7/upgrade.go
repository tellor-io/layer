package v6_1_7

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

/*
Upgrade to v6.1.7 includes (since v6.1.6):
  - Dispute vote validation and accounting (#1061): reject invalid vote choices;
    counted selector stake moves stored ReporterPower on revote (no re-snapshot,
    re-peel, or re-reserve); lock only blocks first acquisition of reporter stake;
    peels target evidence-reporter ownership after a switch when needed; fail-closed
    bucket underflow/overflow; same safeguards applied to team votes.
  - MsgBatchSubmitValue correctness (#1064): align with MsgSubmitValue power caching,
    call ReporterStake on the first verified valid report, and return an error when
    all reports in the batch fail.
  - Reporter self-demotion hardening (#1065): safer demotion lifecycle (reporter not
    deleted immediately), block pending switches onto a demoting reporter, and handle
    rewards/liveness after demotion when the reporter object is gone.
  - Bridge module authority wiring (#1058): inject authority correctly into
    ProvideModule via the module config proto.
  - Bridge vote-extension signing (#1046 / #1063): sign via validated
    checkpoint/attestation RPCs (VoteExtensionSigner), not SignRaw; follow-up signer
    API and registration-sig updates.

No custom state migration is required beyond RunMigrations: dispute vote store
layout and protos are unchanged. In-flight disputes keep existing tallies;
subsequent votes use the corrected accounting.
*/

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info(fmt.Sprintf("Running %s Upgrade...", UpgradeName))

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
