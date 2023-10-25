package keeper_test

import (
	"context"
	"testing"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/mocks"
	keepertest "github.com/tellor-io/layer/testutil/keeper"
	"github.com/tellor-io/layer/x/oracle/keeper"
	"github.com/tellor-io/layer/x/oracle/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, *keeper.Keeper, *mocks.StakingKeeper, *mocks.AccountKeeper, context.Context) {
	k, sk, ak, ctx := keepertest.OracleKeeper(t)
	return keeper.NewMsgServerImpl(*k), k, sk, ak, sdk.WrapSDKContext(ctx)
}

func TestMsgServer(t *testing.T) {
	ms, _, _, _, goctx := setupMsgServer(t)

	require.NotNil(t, ms)
	require.NotNil(t, goctx)
}
func KeyTestPubAddr() (cryptotypes.PrivKey, cryptotypes.PubKey, sdk.AccAddress) {
	key := secp256k1.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}