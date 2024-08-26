package keeper_test

import (
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	math "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgWithdrawTokens(t *testing.T) {
	k, _, bk, ok, _, sk, ctx := setupKeeper(t)
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
		QueryId:              []byte("withdrawTokens"),
		AggregateValue:       "10 * 1e6",
		AggregateReporter:    "reporter1",
		ReporterPower:        int64(100),
		StandardDeviation:    "0",
		Reporters:            []*oracletypes.AggregateReporter{},
		Flagged:              false,
		Index:                uint64(0),
		AggregateReportIndex: int64(0),
		Height:               int64(10_000),
		MicroHeight:          int64(0),
	}
	sk.On("TotalBondedTokens", ctx).Return(math.NewInt(10*1e6), nil)
	// ok.On("SetAggregate", ctx, &aggregate).Return(nil)
	agg, err := k.CreateWithdrawalAggregate(ctx, amount, creatorAddr, []byte(recipientAddr), 1)
	require.NoError(t, err)
	require.NotNil(t, agg)
	ok.On("SetAggregate", ctx, agg).Return(nil)

	response, err = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: "3536373839306162636465663132333435363738",
		Amount:    amount,
	})
	require.NoError(t, err)
	require.NotNil(t, response)
}
