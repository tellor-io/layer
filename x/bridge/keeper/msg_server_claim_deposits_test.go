package keeper_test

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/tellor-io/layer/x/bridge/keeper"
	bridgetypes "github.com/tellor-io/layer/x/bridge/types"
	oracletypes "github.com/tellor-io/layer/x/oracle/types"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestMsgClaimDeposits(t *testing.T) {
	k, _, bk, ok, _, _, _, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	require.Panics(t, func() {
		_, err := msgServer.ClaimDeposits(ctx, nil)
		require.Error(t, err)
	})

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	aggregateTimestamp := sdkCtx.BlockTime()
	AddressType, err := abi.NewType("address", "", nil)
	require.NoError(t, err)
	Uint256Type, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	StringType, err := abi.NewType("string", "", nil)
	require.NoError(t, err)
	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
		{Type: Uint256Type},
	}
	ethAddress := common.HexToAddress("0x3386518F7ab3eb51591571adBE62CF94540EAd29")
	layerAddressString := simtestutil.CreateIncrementalAccounts(1)[0].String()
	amountUint64 := big.NewInt(100 * 1e12)
	tipAmountUint64 := big.NewInt(1 * 1e12)
	reportValueArgsEncoded, err := reportValueArgs.Pack(ethAddress, layerAddressString, amountUint64, tipAmountUint64)
	require.NoError(t, err)
	reportValueString := hex.EncodeToString(reportValueArgsEncoded)
	queryId, err := k.GetDepositQueryId(0)
	require.NoError(t, err)
	aggregate := oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: reportValueString,
		AggregatePower: uint64(68),
	}
	powerThreshold := uint64(67)
	validatorTimestamp := uint64(aggregateTimestamp.UnixMilli() - 1)
	valSetHash := []byte("valSetHash")
	_, err = k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, valSetHash)
	require.NoError(t, err)
	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	// convert ethAddress (common.Address) to sdk.Address
	msgSender := sdk.AccAddress(ethAddress.Bytes())
	recipient, amount, _, err := k.DecodeDepositReportValue(ctx, reportValueString)
	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(aggregateTimestamp.UnixMilli())).Return(aggregate, nil)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, amount).Return(err)

	depositId := uint64(0)

	result, err := msgServer.ClaimDeposits(sdkCtx, &bridgetypes.MsgClaimDepositsRequest{
		Creator:    msgSender.String(),
		DepositIds: []uint64{depositId},
		Timestamps: []uint64{uint64(aggregateTimestamp.UnixMilli())},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	depositClaimedResult, err := k.DepositIdClaimedMap.Get(sdkCtx, depositId)
	require.NoError(t, err)
	require.Equal(t, depositClaimedResult.Claimed, true)
}

func BenchmarkMsgClaimDeposits(b *testing.B) {
	k, _, bk, ok, _, _, _, ctx := setupKeeper(b)
	msgServer := keeper.NewMsgServerImpl(k)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	aggregateTimestamp := sdkCtx.BlockTime()

	AddressType, err := abi.NewType("address", "", nil)
	require.NoError(b, err)
	Uint256Type, err := abi.NewType("uint256", "", nil)
	require.NoError(b, err)
	StringType, err := abi.NewType("string", "", nil)
	require.NoError(b, err)

	reportValueArgs := abi.Arguments{
		{Type: AddressType},
		{Type: StringType},
		{Type: Uint256Type},
		{Type: Uint256Type},
	}

	ethAddress := common.HexToAddress("0x3386518F7ab3eb51591571adBE62CF94540EAd29")
	layerAddressString := simtestutil.CreateIncrementalAccounts(1)[0].String()
	amountUint64 := big.NewInt(100 * 1e12)
	tipAmountUint64 := big.NewInt(1 * 1e12)

	reportValueArgsEncoded, err := reportValueArgs.Pack(ethAddress, layerAddressString, amountUint64, tipAmountUint64)
	require.NoError(b, err)
	reportValueString := hex.EncodeToString(reportValueArgsEncoded)

	queryId, err := k.GetDepositQueryId(0)
	require.NoError(b, err)

	aggregate := oracletypes.Aggregate{
		QueryId:        queryId,
		AggregateValue: reportValueString,
		AggregatePower: uint64(68),
	}

	powerThreshold := uint64(67)
	validatorTimestamp := uint64(aggregateTimestamp.UnixMilli() - 1)
	valSetHash := []byte("valSetHash")
	_, err = k.CalculateValidatorSetCheckpoint(ctx, powerThreshold, validatorTimestamp, valSetHash)
	require.NoError(b, err)

	sdkCtx = sdkCtx.WithBlockTime(sdkCtx.BlockTime().Add(13 * time.Hour))
	msgSender := sdk.AccAddress(ethAddress.Bytes())
	recipient, amount, tip, err := k.DecodeDepositReportValue(ctx, reportValueString)
	claimAmount := amount.Sub(tip...)

	ok.On("GetAggregateByTimestamp", sdkCtx, queryId, uint64(aggregateTimestamp.UnixMilli())).Return(aggregate, nil)
	bk.On("MintCoins", sdkCtx, bridgetypes.ModuleName, amount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, recipient, claimAmount).Return(err)
	bk.On("SendCoinsFromModuleToAccount", sdkCtx, bridgetypes.ModuleName, msgSender, tip).Return(err)

	msg := &bridgetypes.MsgClaimDepositsRequest{
		Creator:    msgSender.String(),
		DepositIds: []uint64{0},
		Timestamps: []uint64{uint64(aggregateTimestamp.UnixMilli())},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		err = k.DepositIdClaimedMap.Remove(sdkCtx, uint64(0))
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		result, err := msgServer.ClaimDeposits(sdkCtx, msg)
		if err != nil {
			b.Fatal(err)
		}
		if result == nil {
			b.Fatal("result should not be nil")
		}
	}
}
