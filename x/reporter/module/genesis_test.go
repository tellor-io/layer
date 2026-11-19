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

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	genesisState.TwoBitReporters = append(genesisState.TwoBitReporters, &types.TwoBitReporterStateEntry{AgentAddress: []byte("agent"), ReporterAddress: []byte("reporter")})

	k, _, _, _, _, ctx, _ = keepertest.ReporterKeeper(t)
	ctx = ctx.WithBlockHeight(1134000 + 100)
	require.NotPanics(t, func() { reporter.InitGenesis(ctx, k, genesisState) })
	got2 := reporter.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	require.Equal(t, genesisState.SelectorTips, got2.SelectorTips)
	require.Equal(t, genesisState.DisputedDelegationAmounts, got2.DisputedDelegationAmounts)
	require.Equal(t, genesisState.FeePaidFromStake, got2.FeePaidFromStake)
	require.Equal(t, genesisState.TwoBitReporters, got2.TwoBitReporters)

	err := os.Remove("reporter_module_state.json")
	require.NoError(t, err)

	// this line is used by starport scaffolding # genesis/test/assert
}

func TestSelectorSwitchHistoryGenesisRoundTrip(t *testing.T) {
	k, _, _, _, _, ctx, _ := keepertest.ReporterKeeper(t)
	selector := testAcc(1)
	openOriginal := testAcc(2)
	olderOriginal := testAcc(3)
	require.NoError(t, k.Selectors.Set(ctx, selector.Bytes(), types.NewSelection(openOriginal, 1)))
	history := types.SelectorReporterSwitches{
		Switches: []types.SelectorReporterSwitch{
			{OriginalReporter: openOriginal.Bytes(), BeganHeight: 300, CompletedHeight: 0},
			{OriginalReporter: olderOriginal.Bytes(), BeganHeight: 100, CompletedHeight: 200},
		},
	}
	require.NoError(t, k.SelectorReporterSwitches.Set(ctx, selector.Bytes(), history))
	require.NoError(t, k.SelectorSwitchPruneCursor.Set(ctx, selector.Bytes()))

	type heightAnswer struct {
		reporter   sdk.AccAddress
		live       bool
		unfinished bool
	}
	heights := []uint64{50, 150, 250, 350}
	before := make([]heightAnswer, len(heights))
	for i, h := range heights {
		got, live, err := k.ReporterAtHeight(ctx, selector, h)
		require.NoError(t, err)
		unfinished, err := k.SelectorSwitchUnfinishedAt(ctx, selector, h)
		require.NoError(t, err)
		before[i] = heightAnswer{reporter: got, live: live, unfinished: unfinished}
	}

	exported := reporter.ExportGenesis(ctx, k)
	require.Len(t, exported.SelectorReporterSwitches, 1)
	require.Equal(t, selector.Bytes(), exported.SelectorReporterSwitches[0].SelectorAddress)
	require.Len(t, exported.SelectorReporterSwitches[0].Switches, 2)
	require.Zero(t, exported.SelectorReporterSwitches[0].Switches[0].CompletedHeight)
	require.Equal(t, uint64(200), exported.SelectorReporterSwitches[0].Switches[1].CompletedHeight)

	k2, _, _, _, _, ctx2, _ := keepertest.ReporterKeeper(t)
	require.NoError(t, k2.Selectors.Set(ctx2, selector.Bytes(), types.NewSelection(openOriginal, 1)))
	require.NotPanics(t, func() { reporter.InitGenesis(ctx2, k2, *exported) })

	imported, err := k2.SelectorReporterSwitches.Get(ctx2, selector.Bytes())
	require.NoError(t, err)
	require.Equal(t, history.Switches, imported.Switches)
	hasCursor, err := k2.SelectorSwitchPruneCursor.Has(ctx2)
	require.NoError(t, err)
	require.False(t, hasCursor)

	for i, h := range heights {
		got, live, err := k2.ReporterAtHeight(ctx2, selector, h)
		require.NoError(t, err)
		unfinished, err := k2.SelectorSwitchUnfinishedAt(ctx2, selector, h)
		require.NoError(t, err)
		require.Equal(t, []byte(before[i].reporter), []byte(got))
		require.Equal(t, before[i].live, live)
		require.Equal(t, before[i].unfinished, unfinished)
	}

	err = os.Remove("reporter_module_state.json")
	require.NoError(t, err)
}

func testAcc(last byte) sdk.AccAddress {
	addr := make([]byte, 20)
	addr[19] = last
	return addr
}
