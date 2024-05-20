package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testkeeper "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func TestSetAndGetParams(t *testing.T) {
	k, _, _, ctx := testkeeper.RegistryKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)
	p, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.EqualValues(t, params, p)
}
