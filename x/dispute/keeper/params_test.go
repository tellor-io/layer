package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/dispute/types"
)

func TestGetParams(t *testing.T) {
	k, _, _, _, _, ctx := testkeeper.DisputeKeeper(t)
	params := types.DefaultParams()

	k.Params.Set(ctx, params)

	params, err := k.Params.Get(ctx)
	require.NoError(t, err)
	require.EqualValues(t, params, params)
}
