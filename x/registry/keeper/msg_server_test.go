package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context, keeper.Keeper) {
	k, ctx := keepertest.RegistryKeeper(t)
	return keeper.NewMsgServerImpl(k), sdk.WrapSDKContext(ctx), k
}

func TestMsgServer(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
}
