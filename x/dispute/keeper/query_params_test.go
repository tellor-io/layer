package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/dispute/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
)

func TestParamsQuery(t *testing.T) {
	k, _, _, _, _, ctx := testkeeper.DisputeKeeper(t)
	q := keeper.NewQuerier(k)
	params := types.DefaultParams()
	k.Params.Set(ctx, params)

	response, err := q.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, &types.QueryParamsResponse{Params: params}, response)
}
