package integration_test

import (
	"github.com/tellor-io/layer/utils"
	minttypes "github.com/tellor-io/layer/x/mint/types"
	"github.com/tellor-io/layer/x/oracle"
	reporterkeeper "github.com/tellor-io/layer/x/reporter/keeper"
	reportertypes "github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestSelfDemotionThenLivenessDistributionEndBlocker verifies self-demotion does not
// halt liveness payout:
//
//  1. Reporter A earns positive liveness share in the current period.
//  2. A self-demotes to B (Reporters[A] is retained until the pending switch finalizes).
//  3. Liveness distribution (oracle EndBlocker → RotateQueries →
//     CheckAndDistributeLivenessRewards → DistributeLivenessRewards →
//     AllocateTBR → DivvyingTips) completes without error.
func (s *IntegrationTestSuite) TestSelfDemotionThenLivenessDistributionEndBlocker() {
	require := s.Require()
	ctx := s.Setup.Ctx

	reporterA, reporterB, _, _ := s.reporterSelfSwitchFixture()
	msgServer := reporterkeeper.NewMsgServerImpl(s.Setup.Reporterkeeper)

	tipper := s.newKeysWithTokens()
	tbrAmount := math.NewInt(100_000)
	require.NoError(s.Setup.Bankkeeper.SendCoinsFromAccountToModule(
		ctx, tipper, minttypes.TimeBasedRewards,
		sdk.NewCoins(sdk.NewCoin(s.Setup.Denom, tbrAmount)),
	))

	cyclelist, err := s.Setup.Oraclekeeper.GetCyclelist(ctx)
	require.NoError(err)
	require.NotEmpty(cyclelist)

	require.NoError(s.Setup.Oraclekeeper.ResetLivenessData(ctx))
	require.NoError(s.Setup.Oraclekeeper.IncrementStandardOpportunities(ctx))
	for _, queryData := range cyclelist {
		queryId := utils.QueryIDFromData(queryData)
		require.NoError(s.Setup.Oraclekeeper.IncrementQueryOpportunities(ctx, queryId))
		// A earned share this period (reports already aggregated before demotion).
		require.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, reporterA.Bytes(), queryId, 100, 100, false))
		require.NoError(s.Setup.Oraclekeeper.UpdateReporterLiveness(ctx, reporterB.Bytes(), queryId, 100, 100, false))
	}

	shareA, err := s.Setup.Oraclekeeper.ReporterStandardShareSum.Get(ctx, reporterA.Bytes())
	require.NoError(err)
	require.True(shareA.IsPositive(), "precondition: A must have retained liveness allocation")

	// No real open commitments (liveness was injected). Advance past height 0 so the
	// maxCommit >= currentBlock guard does not treat a zero maxCommit as still open.
	maxCommit, err := s.Setup.Oraclekeeper.GetMaxOpenCommitmentForReporter(ctx, reporterA.Bytes())
	require.NoError(err)
	ctx = ctx.WithBlockHeight(int64(maxCommit) + 1)
	s.Setup.Ctx = ctx

	_, err = msgServer.SwitchReporter(ctx, &reportertypes.MsgSwitchReporter{
		SelectorAddress: reporterA.String(),
		ReporterAddress: reporterB.String(),
	})
	require.NoError(err)

	// Direct payout used by the EndBlocker path above. This is the bug trigger.
	err = s.Setup.Oraclekeeper.DistributeLivenessRewards(ctx)
	require.NoError(err, "liveness distribution must succeed after self-demotion (avoids oracle EndBlocker halt)")

	// EndBlocker itself must also remain healthy after the demotion.
	require.NoError(oracle.EndBlocker(ctx, s.Setup.Oraclekeeper))
}
