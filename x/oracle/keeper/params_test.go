package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "layer/testutil/keeper"
	"layer/x/oracle/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.OracleKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
