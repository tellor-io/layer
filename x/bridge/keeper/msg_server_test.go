package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
)

func setupMsgServer(tb testing.TB) types.MsgServer {
	tb.Helper()
	k, _, _, _, _, _, _ := keepertest.BridgeKeeper(tb)
	return keeper.NewMsgServerImpl(k)
}

func TestMsgServer(t *testing.T) {
	ms := setupMsgServer(t)
	require.NotNil(t, ms)
}