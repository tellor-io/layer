package keeper_test

import (
	"encoding/hex"
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	math "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgWithdrawTokens(t *testing.T) {
	k, _, bk, ok, _, sk, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	msgServer := keeper.NewMsgServerImpl(k)

	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	recipientAddr := "1234567890abcdef1234567890abcdef12345678"

	// amount is zero
	response, err := msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: recipientAddr,
		Amount:    sdk.Coin{Denom: "loya", Amount: math.NewInt(0)},
	})
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, response)

	// denom is not loya
	response, err = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: recipientAddr,
		Amount:    sdk.Coin{Denom: "eth", Amount: math.NewInt(10 * 1e6)},
	})
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, response)

	// amount is negative
	response, err = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: recipientAddr,
		Amount:    sdk.Coin{Denom: "loya", Amount: math.NewInt(-10 * 1e6)},
	})
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, response)

	amount := sdk.Coin{Denom: "loya", Amount: math.NewInt(10 * 1e6)}
	bk.On("SendCoinsFromAccountToModule", ctx, creatorAddr, types.ModuleName, sdk.NewCoins(amount)).Return(nil)
	bk.On("BurnCoins", ctx, types.ModuleName, sdk.NewCoins(amount)).Return(nil)
	_ = oracletypes.Aggregate{
		QueryId:           []byte("withdrawTokens"),
		AggregateValue:    "10 * 1e6",
		AggregateReporter: "reporter1",
		AggregatePower:    uint64(100),
		Flagged:           false,
		Index:             uint64(0),
		Height:            uint64(10_000),
		MicroHeight:       uint64(0),
	}
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(10*1e6), nil)
	// ok.On("SetAggregate", ctx, &aggregate).Return(nil)
	agg, queryData, err := k.CreateWithdrawalAggregate(ctx, amount, creatorAddr, []byte(recipientAddr), 1)
	require.NoError(t, err)
	require.NotNil(t, agg)
	require.NotNil(t, queryData)
	ok.On("SetAggregate", ctx, agg, queryData, mock.AnythingOfType("string")).Return(nil)

	response, err = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: "3536373839306162636465663132333435363738",
		Amount:    amount,
	})
	require.NoError(t, err)
	require.NotNil(t, response)
}

func TestMsgWithdrawTokensBadRecipient(t *testing.T) {
	k, _, bk, ok, _, sk, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	msgServer := keeper.NewMsgServerImpl(k)

	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	badRecipientInvalidHex := "z1234567890abcdef1234567890abcdef1234567"
	badRecipientInvalidLength := "1234567890abcdef1234567890abcdef123456"
	goodRecipientAddr := "1234567890abcdef1234567890abcdef12345679"

	// bad recipient invalid hex
	response, err := msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: badRecipientInvalidHex,
		Amount:    sdk.Coin{Denom: "loya", Amount: math.NewInt(10 * 1e6)},
	})
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, response)

	// bad recipient invalid length
	response, err = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: badRecipientInvalidLength,
		Amount:    sdk.Coin{Denom: "loya", Amount: math.NewInt(10 * 1e6)},
	})
	require.ErrorContains(t, err, "invalid request")
	require.Nil(t, response)

	// good recipient
	amount := sdk.Coin{Denom: "loya", Amount: math.NewInt(10 * 1e6)}
	bk.On("SendCoinsFromAccountToModule", ctx, creatorAddr, types.ModuleName, sdk.NewCoins(amount)).Return(nil)
	bk.On("BurnCoins", ctx, types.ModuleName, sdk.NewCoins(amount)).Return(nil)
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(10*1e6), nil)
	goodRecipientHex, err := hex.DecodeString(goodRecipientAddr)
	require.NoError(t, err)
	agg, queryData, err := k.CreateWithdrawalAggregate(ctx, amount, creatorAddr, goodRecipientHex, 1)
	require.NoError(t, err)
	require.NotNil(t, agg)
	require.NotNil(t, queryData)
	ok.On("SetAggregate", ctx, agg, queryData, mock.AnythingOfType("string")).Return(nil)
	response, err = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: goodRecipientAddr,
		Amount:    amount,
	})
	require.NoError(t, err)
	require.NotNil(t, response)
}

func BenchmarkMsgWithdrawTokens(b *testing.B) {
	k, _, bk, ok, _, sk, _, ctx := setupKeeper(b)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(k)
	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	recipientAddr := "3536373839306162636465663132333435363738"
	amount := sdk.Coin{Denom: "loya", Amount: math.NewInt(10 * 1e6)}

	// setup mock expectations
	bk.On("SendCoinsFromAccountToModule", sdkCtx, creatorAddr, types.ModuleName, sdk.NewCoins(amount)).Return(nil)
	bk.On("BurnCoins", sdkCtx, types.ModuleName, sdk.NewCoins(amount)).Return(nil)
	sk.On("TotalBondedTokens", sdkCtx).Return(math.NewInt(10*1e6), nil)

	// Use AnythingOfType to match any aggregate and query data
	ok.On("SetAggregate",
		mock.AnythingOfType("types.Context"),
		mock.AnythingOfType("*types.Aggregate"),
		mock.AnythingOfType("[]uint8"),
		mock.AnythingOfType("string")).Return(nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
			Creator:   creatorAddr.String(),
			Recipient: recipientAddr,
			Amount:    amount,
		})
	}
}
