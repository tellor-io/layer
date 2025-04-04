package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func TestParamsQuery(t *testing.T) {
	k, _, _, _, _, _, _, ctx := testkeeper.BridgeKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, k.Params.Set(ctx, params))

	response, err := keeper.NewQuerier(k).Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
