package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
	"github.com/tellor-io/layer/x/reporter/types"
)

func setupMsgServer(tb testing.TB) (keeper.Keeper, *mocks.StakingKeeper, *mocks.BankKeeper, types.MsgServer, context.Context) {
	tb.Helper()
	k, sk, bk, ctx := setupKeeper(tb)
	return k, sk, bk, keeper.NewMsgServerImpl(k), ctx
}

func TestMsgServer(t *testing.T) {
	k, sk, bk, ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
	require.NotNil(t, sk)
	require.NotNil(t, bk)
}
