package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/mint/types"
)

func TestGenesis(t *testing.T) {
	require := require.New(t)
	k, ak, _, ctx := keepertest.MintKeeper(t)

	time := time.Now()
	genesisState := types.NewGenesisState("loya", true, &time)
	require.NotNil(genesisState)
	ak.On("GetModuleAccount", ctx, types.TimeBasedRewards).Return(nil)
	ak.On("GetModuleAccount", ctx, types.ExtraRewardsPool).Return(nil)
	k.InitGenesis(ctx, ak, genesisState)
	got := k.ExportGenesis(ctx)
	require.NotNil(got)
	require.Equal(got.BondDenom, "loya")
	require.True(got.Initialized)
	require.NotNil(got.PreviousBlockTime)
}
