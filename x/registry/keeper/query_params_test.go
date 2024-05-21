package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	rk "github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestParamsQuery(t *testing.T) {
	keeper, _, _, ctx := testkeeper.RegistryKeeper(t)
	wctx := sdk.WrapSDKContext(ctx)
	params := types.DefaultParams()
	keeper.SetParams(ctx, params)
	queryier := rk.NewQuerier(keeper)
	response, err := queryier.Params(wctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
