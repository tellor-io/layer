package keeper_test

import (
	"testing"

	math "cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"
)

func TestMsgWithdrawTokens(t *testing.T) {
	msgServer, ctx := setupMsgServer(t)
	require.NotNil(t, msgServer)
	require.NotNil(t, ctx)
	k, _, bk, ok, _, _, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)

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
	bk.On("SendCoinsFromAccountToModule", ctx, creatorAddr, types.ModuleName, sdk.NewCoins(amount)).Return(nil).Once()
	// bk.On("SendCoinsFromAccountToModule", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	bk.On("BurnCoins", ctx, types.ModuleName, sdk.NewCoins(amount)).Return(nil).Once()
	aggregate := oracletypes.Aggregate{
		QueryId:              []byte("withdrawTokens"),
		AggregateValue:       "10 * 1e6",
		AggregateReporter:    "reporter1",
		ReporterPower:        int64(100),
		StandardDeviation:    float64(0),
		Reporters:            []*oracletypes.AggregateReporter{{}},
		Flagged:              false,
		Nonce:                uint64(0),
		AggregateReportIndex: int64(0),
		Height:               int64(0),
		MicroHeight:          int64(0),
	}
	ok.On("SetAggregate", ctx, aggregate).Return(nil).Once()

	response, err = msgServer.WithdrawTokens(ctx, &types.MsgWithdrawTokens{
		Creator:   creatorAddr.String(),
		Recipient: recipientAddr,
		Amount:    amount,
	})
	require.NoError(t, err)
	require.NotNil(t, response)
}
