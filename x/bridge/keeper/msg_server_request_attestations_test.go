package keeper_test

import (
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	"github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	math "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestMsgRequestAttestations(t *testing.T) {
	k, _, _, ok, _, sk, ctx := setupKeeper(t)
	require.NotNil(t, k)
	require.NotNil(t, ctx)
	msgServer := keeper.NewMsgServerImpl(k)
	require.NotNil(t, msgServer)

	// empty message
	response, err := msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{})
	require.ErrorContains(t, err, "InvalidArgument")
	require.Nil(t, response)

	// bad queryId
	response, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   "abcd1234",
		QueryId:   "z",
		Timestamp: "1",
	})
	require.ErrorContains(t, err, "InvalidArgument")
	require.Nil(t, response)

	// bad timestamp
	response, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   "abcd1234",
		QueryId:   "abcd1234",
		Timestamp: "z",
	})
	require.ErrorContains(t, err, "InvalidArgument")
	require.Nil(t, response)

	validators := []stakingtypes.Validator{
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(60 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator1"},
			OperatorAddress: "operatorAddr1",
		},
		{
			Jailed:          false,
			Status:          stakingtypes.Bonded,
			Tokens:          math.NewInt(40 * 1e6),
			DelegatorShares: math.LegacyNewDec(100 * 1e6),
			Description:     stakingtypes.Description{Moniker: "validator2"},
			OperatorAddress: "operatorAddr2",
		},
	}

	evmAddresses := make([]types.EVMAddress, len(validators))
	for i, val := range validators {
		err := k.SetEVMAddressByOperator(ctx, val.OperatorAddress, []byte(val.Description.Moniker))
		require.NoError(t, err)

		evmAddresses[i], err = k.OperatorToEVMAddressMap.Get(ctx, val.GetOperator())
		require.NoError(t, err)
		require.Equal(t, evmAddresses[i].EVMAddress, []byte(val.Description.Moniker))
	}
	sk.On("GetAllValidators", ctx).Return(validators, nil)
	result, err := k.CompareAndSetBridgeValidators(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)

	creatorAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	queryId := []byte("83a7f3d48786ac2667503a61e8c415438ed2922eb86a2906e4ee66d9a2ce499")
	timestampTime := time.Unix(1000, 0)
	aggReport := oracletypes.Aggregate{
		QueryId:           queryId,
		AggregateValue:    "10",
		ReporterPower:     int64(10),
		AggregateReporter: creatorAddr.String(),
	}
	ok.On("GetAggregateByTimestamp", ctx, queryId, timestampTime).Return(&aggReport, nil)
	err = k.ValidatorCheckpoint.Set(ctx, types.ValidatorCheckpoint{
		Checkpoint: []byte("checkpoint"),
	})
	require.NoError(t, err)

	ok.On("GetTimestampBefore", ctx, queryId, timestampTime).Return(timestampTime.Add(-1*time.Hour), nil)
	ok.On("GetTimestampAfter", ctx, queryId, timestampTime).Return(timestampTime.Add(1*time.Hour), nil)

	response, err = msgServer.RequestAttestations(ctx, &types.MsgRequestAttestations{
		Creator:   creatorAddr.String(),
		QueryId:   hex.EncodeToString(queryId),
		Timestamp: strconv.FormatInt(timestampTime.UnixMilli(), 10),
	})
	require.NoError(t, err)
	require.NotNil(t, response)
}
