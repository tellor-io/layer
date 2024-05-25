package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

func setupMsgServer(tb testing.TB) (types.MsgServer, context.Context, keeper.Keeper) {
	tb.Helper()
	k, _, _, ctx := keepertest.RegistryKeeper(tb)
	return keeper.NewMsgServerImpl(k), ctx, k
}

func TestMsgServer(t *testing.T) {
	ms, ctx, k := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
}
