package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMsgClaimDeposit(t *testing.T) {
	msgServer, ctx := setupMsgServer(t)
	require.NotNil(t, msgServer)
	require.NotNil(t, ctx)
	k, ak, bk, ok, rk, sk, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ak)
	require.NotNil(t, bk)
	require.NotNil(t, ok)
	require.NotNil(t, rk)
	require.NotNil(t, sk)
	require.NotNil(t, ctx)

	// creator := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	// msgClaimDeposit := &types.MsgClaimDepositRequest{
	// 	Creator:   creator.String(),
	// 	DepositId: uint64(1),
	// 	Index:     reportIndex,
	// }

	// _, err = msgServer.ClaimDeposit(ctx, msgClaimDeposit)
	// require.NoError(t, err)
}
