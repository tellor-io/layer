package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	rk "github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, _, _, ctx := testkeeper.RegistryKeeper(t)
	querier := rk.NewQuerier(keeper)

	res, err := querier.Params(ctx, nil)
	require.Error(t, err)
	require.Nil(t, res)

	params := types.DefaultParams()
	require.NoError(t, keeper.SetParams(ctx, params))
	response, err := querier.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
