package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, _, _, _, _, _, ctx := keepertest.BridgeKeeper(t)
	return keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
}
