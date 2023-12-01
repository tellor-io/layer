package keeper_test

import (
	"context"
	"testing"

	storeTypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/registry/keeper"
	"github.com/tellor-io/layer/x/registry/types"
)

// helper function that gets used by all registry keeper tests
// messages are the way that tx interact with the blockchain
// each msg is associated w/ a specific module, and is intended to trigger something in the module
func setupMsgServer(t testing.TB) (types.MsgServer, context.Context, keeper.Keeper, storeTypes.KVStoreKey) {
	k, ctx, key := keepertest.RegistryKeeper(t)
	// returns a MsgServerImpl struct, sdk.Context, keeper.Keeper, and storeTypes.KVStoreKey
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx), *k, key
}

func TestMsgServer(t *testing.T) {
	ms, ctx, k, key := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
	require.NotNil(t, k)
	require.NotNil(t, key)
}
