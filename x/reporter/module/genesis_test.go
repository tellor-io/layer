package reporter_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/nullify"
	reporter "github.com/tellor-io/layer/x/reporter/module"
	"github.com/tellor-io/layer/x/reporter/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:                    types.DefaultParams(),
		Reporters:                 []*types.ReporterStateEntry{},
		SelectorTips:              []*types.SelectorTipsStateEntry{},
		Selectors:                 []*types.SelectorsStateEntry{},
		DisputedDelegationAmounts: []*types.DisputedDelegationAmountStateEntry{},
		FeePaidFromStake:          []*types.FeePaidFromStakeStateEntry{},
		Report:                    []*types.ReportStateEntry{},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, _, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	require.NotPanics(t, func() { reporter.InitGenesis(ctx, k, genesisState) })
	got := reporter.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	genesisState.Reporters = append(genesisState.Reporters, &types.ReporterStateEntry{ReporterAddress: []byte("reporter"), Reporter: &types.OracleReporter{MinTokensRequired: math.NewInt(1_000_000), Moniker: "caleb", Jailed: false, CommissionRate: math.LegacyMustNewDecFromStr("0.2")}})
	genesisState.SelectorTips = append(genesisState.SelectorTips, &types.SelectorTipsStateEntry{SelectorAddress: []byte("selector"), Tips: math.LegacyMustNewDecFromStr("20000")})
	genesisState.Selectors = append(genesisState.Selectors, &types.SelectorsStateEntry{SelectorAddress: []byte("selector"), Selection: &types.Selection{Reporter: []byte("reporter"), DelegationsCount: 1}})
	genesisState.DisputedDelegationAmounts = append(genesisState.DisputedDelegationAmounts, &types.DisputedDelegationAmountStateEntry{HashId: []byte("hash id"), DelegationAmount: &types.DelegationsAmounts{TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: []byte("delegator"), ValidatorAddress: []byte("val"), Amount: math.NewInt(1_000_000)}}, Total: math.NewInt(1_000_000)}})
	genesisState.FeePaidFromStake = append(genesisState.FeePaidFromStake, &types.FeePaidFromStakeStateEntry{HashId: []byte("hash id"), DelegationAmount: &types.DelegationsAmounts{TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: []byte("delegator"), ValidatorAddress: []byte("val"), Amount: math.NewInt(1_000_000)}}, Total: math.NewInt(1_000_000)}})
	genesisState.Report = append(genesisState.Report, &types.ReportStateEntry{QueryId: []byte("query_id"), ReporterAddress: []byte("reporter"), BlockHeight: 10, DelegationAmount: &types.DelegationsAmounts{TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: []byte("delegator"), ValidatorAddress: []byte("val"), Amount: math.NewInt(1_000_000)}}, Total: math.NewInt(1_000_000)}})
	genesisState.Report = append(genesisState.Report, &types.ReportStateEntry{QueryId: []byte("query_id"), ReporterAddress: []byte("reporter"), BlockHeight: 1134000, DelegationAmount: &types.DelegationsAmounts{TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: []byte("delegator"), ValidatorAddress: []byte("val"), Amount: math.NewInt(1_000_000)}}, Total: math.NewInt(1_000_000)}})

	k, _, _, _, _, ctx, _ = keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1134000 + 100)
	require.NotPanics(t, func() { reporter.InitGenesis(ctx, k, genesisState) })
	got2 := reporter.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	require.Equal(t, genesisState.Reporters, got2.Reporters)
	require.Equal(t, genesisState.SelectorTips, got2.SelectorTips)
	require.Equal(t, genesisState.Selectors, got2.Selectors)
	require.Equal(t, genesisState.DisputedDelegationAmounts, got2.DisputedDelegationAmounts)
	require.Equal(t, genesisState.FeePaidFromStake, got2.FeePaidFromStake)
	require.Equal(t, []*types.ReportStateEntry{genesisState.Report[1]}, got2.Report)

	// this line is used by starport scaffolding # genesis/test/assert
}
