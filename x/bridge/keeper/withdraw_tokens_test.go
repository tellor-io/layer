package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/types"

	math "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestGetWithdrawalReportValue(t *testing.T) {
	k, _, _, _, _, _, _ := setupKeeper(t)

	res, err := k.GetWithdrawalReportValue(sdk.Coin{Amount: math.NewInt(100), Denom: "loya"}, sdk.AccAddress("operatorAddr1"), []byte("evmAddress1"))
	require.NoError(t, err)
	require.NotNil(t, res)

	res2, err := k.GetWithdrawalReportValue(sdk.Coin{Amount: math.NewInt(100), Denom: "loya"}, sdk.AccAddress("operatorAddr2"), []byte("evmAddress2"))
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.NotEqual(t, res, res2)

	res3, err := k.GetWithdrawalReportValue(sdk.Coin{Amount: math.NewInt(100), Denom: "loya"}, sdk.AccAddress("operatorAddr1"), []byte("evmAddress1"))
	require.NoError(t, err)
	require.NotNil(t, res3)
	require.Equal(t, res, res3)
}

func TestGetWithdrawalQueryId(t *testing.T) {
	k, _, _, _, _, _, _ := setupKeeper(t)

	res, queryData, err := k.GetWithdrawalQueryId(1)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, queryData)

	res2, queryData2, err := k.GetWithdrawalQueryId(2)
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.NotEqual(t, res, res2)
	require.NotEqual(t, queryData, queryData2)
}

func TestCreateWithdrawalAggregate(t *testing.T) {
	k, _, _, _, _, sk, ctx := setupKeeper(t)

	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100), nil).Once()
	agg, _, err := k.CreateWithdrawalAggregate(ctx, sdk.Coin{Amount: math.NewInt(100), Denom: "loya"}, sdk.AccAddress("operatorAddr1"), []byte("evmAddress1"), 1)
	require.NoError(t, err)
	require.Equal(t, agg.AggregatePower, uint64(100))
	require.Equal(t, agg.Height, uint64(0))
	require.Equal(t, agg.Flagged, false)
	require.Equal(t, agg.Index, uint64(0))
	queryIdExpected, _, err := k.GetWithdrawalQueryId(1)
	require.NoError(t, err)
	require.Equal(t, agg.QueryId, queryIdExpected)
	aggValueExpected, err := k.GetWithdrawalReportValue(sdk.Coin{Amount: math.NewInt(100), Denom: "loya"}, sdk.AccAddress("operatorAddr1"), []byte("evmAddress1"))
	require.NoError(t, err)
	require.Equal(t, agg.AggregateValue, hex.EncodeToString(aggValueExpected))
}

func TestIncrementWithdrawalId(t *testing.T) {
	k, _, _, _, _, _, ctx := setupKeeper(t)

	id, err := k.IncrementWithdrawalId(ctx)
	require.NoError(t, err)
	require.Equal(t, id, uint64(1))

	id, err = k.IncrementWithdrawalId(ctx)
	require.NoError(t, err)
	require.Equal(t, id, uint64(2))
}

func TestWithdrawTokens(t *testing.T) {
	k, _, bk, ok, _, sk, ctx := setupKeeper(t)

	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	recipientAddr := "1234567890abcdef1234567890abcdef12345678"
	amount := sdk.Coin{Denom: "loya", Amount: math.NewInt(10 * 1e6)}

	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(100*1e6), nil)
	agg, queryData, err := k.CreateWithdrawalAggregate(ctx, amount, creatorAddr, []byte(recipientAddr), 1)
	require.NoError(t, err)
	require.NotNil(t, agg)

	ok.On("SetAggregate", ctx, agg, queryData).Return(nil)
	bk.On("SendCoinsFromAccountToModule", ctx, creatorAddr, types.ModuleName, sdk.NewCoins(amount)).Return(nil)
	bk.On("BurnCoins", ctx, types.ModuleName, sdk.NewCoins(amount)).Return(nil)

	withdrawId, err := k.WithdrawTokens(ctx, amount, creatorAddr, []byte(recipientAddr))
	require.NoError(t, err)
	require.Equal(t, withdrawId, uint64(1))
}
