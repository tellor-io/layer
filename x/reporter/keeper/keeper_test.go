package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/reporter/keeper"
	"github.com/tellor-io/layer/x/reporter/mocks"
)

func setupKeeper(t testing.TB) (keeper.Keeper, *mocks.StakingKeeper, *mocks.BankKeeper, context.Context) {
	k, sk, bk, ctx := keepertest.ReporterKeeper(t)
	return k, sk, bk, ctx
}

func TestKeeper(t *testing.T) {
	k, sk, bk, ctx := keepertest.ReporterKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
	require.NotNil(t, sk)
	require.NotNil(t, bk)
}
