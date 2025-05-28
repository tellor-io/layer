package reporter_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/testutil/nullify"
	reporter "github.com/tellor-io/layer/x/reporter/module"
	"github.com/tellor-io/layer/x/reporter/types"

	"cosmossdk.io/math"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:                    types.DefaultParams(),
		SelectorTips:              []*types.SelectorTipsStateEntry{},
		DisputedDelegationAmounts: []*types.DisputedDelegationAmountStateEntry{},
		FeePaidFromStake:          []*types.FeePaidFromStakeStateEntry{},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, _, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	require.NotPanics(t, func() { reporter.InitGenesis(ctx, k, genesisState) })
	got := reporter.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	genesisState.SelectorTips = append(genesisState.SelectorTips, &types.SelectorTipsStateEntry{SelectorAddress: []byte("selector"), Tips: math.LegacyMustNewDecFromStr("20000")})
	genesisState.DisputedDelegationAmounts = append(genesisState.DisputedDelegationAmounts, &types.DisputedDelegationAmountStateEntry{HashId: []byte("hash id"), DelegationAmount: &types.DelegationsAmounts{TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: []byte("delegator"), ValidatorAddress: []byte("val"), Amount: math.NewInt(1_000_000)}}, Total: math.NewInt(1_000_000)}})
	genesisState.FeePaidFromStake = append(genesisState.FeePaidFromStake, &types.FeePaidFromStakeStateEntry{HashId: []byte("hash id"), DelegationAmount: &types.DelegationsAmounts{TokenOrigins: []*types.TokenOriginInfo{{DelegatorAddress: []byte("delegator"), ValidatorAddress: []byte("val"), Amount: math.NewInt(1_000_000)}}, Total: math.NewInt(1_000_000)}})

	k, _, _, _, _, ctx, _ = keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1134000 + 100)
	require.NotPanics(t, func() { reporter.InitGenesis(ctx, k, genesisState) })
	got2 := reporter.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	require.Equal(t, genesisState.SelectorTips, got2.SelectorTips)
	require.Equal(t, genesisState.DisputedDelegationAmounts, got2.DisputedDelegationAmounts)
	require.Equal(t, genesisState.FeePaidFromStake, got2.FeePaidFromStake)

	err := os.Remove("reporter_module_state.json")
	require.NoError(t, err)

	// this line is used by starport scaffolding # genesis/test/assert
}
