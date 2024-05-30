package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/mint/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.NewGenesisState("loya")
	k, ak, _, ctx := keepertest.MintKeeper(t)
	k.InitGenesis(ctx, ak, genesisState)
	got := k.ExportGenesis(ctx)
	require.NotNil(t, got)
	require.Equal(t, got.BondDenom, "loya")
}
